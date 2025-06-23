package search

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v8/esutil"
	"github.com/gofrs/uuid"
	json "github.com/json-iterator/go"
	"go.uber.org/zap"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/utils/message"
)

const (
	syncInterval    = 1 * time.Minute
	syncMessageBulk = 250
)

type attributes struct {
	To             []uuid.UUID
	Citation       []uuid.UUID
	OgpContent	   []string
	HasURL         bool
	HasAttachments bool
	HasImage       bool
	HasVideo       bool
	HasAudio       bool
}

// ユーザーがbotかどうかのcache
type userCache map[uuid.UUID]bool

// convertMessageCreated 新規メッセージをesへ入れる型に変換する
func (e *esEngine) convertMessageCreated(m *model.Message, parseResult *message.ParseResult, userCache userCache) (*esMessageDoc, error) {
	var isBot, ok bool
	if isBot, ok = userCache[m.UserID]; !ok {
		// 新規ユーザー or キャッシュが存在しない
		user, err := e.repo.GetUser(m.UserID, false)
		if err != nil {
			return nil, err
		}
		isBot = user.IsBot()
	}

	attr, err := e.getAttributes(m, parseResult)
	if err != nil {
		return nil, fmt.Errorf("failed to get attributes: %w", err)
	}

	return &esMessageDoc{
		UserID:         m.UserID,
		ChannelID:      m.ChannelID,
		IsPublic:       e.cm.IsPublicChannel(m.ChannelID),
		Bot:            isBot,
		Text:           m.Text,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
		To:             attr.To,
		Citation:       attr.Citation,
		OgpContent:     attr.OgpContent,
		HasURL:         attr.HasURL,
		HasAttachments: attr.HasAttachments,
		HasImage:       attr.HasImage,
		HasVideo:       attr.HasVideo,
		HasAudio:       attr.HasAudio,
	}, nil
}

// convertMessageUpdated 既存メッセージの更新情報をesへ入れる型に変換する
func (e *esEngine) convertMessageUpdated(m *model.Message, parseResult *message.ParseResult) (*esMessageDocUpdate, error) {
	attr, err := e.getAttributes(m, parseResult)
	if err != nil {
		return nil, fmt.Errorf("failed to get attributes: %w", err)
	}
	// Updateする項目のみ
	return &esMessageDocUpdate{
		Text:           m.Text,
		UpdatedAt:      m.UpdatedAt,
		Citation:       attr.Citation,
		OgpContent:     attr.OgpContent,
		HasURL:         attr.HasURL,
		HasAttachments: attr.HasAttachments,
		HasImage:       attr.HasImage,
		HasVideo:       attr.HasVideo,
		HasAudio:       attr.HasAudio,
	}, nil
}

func (e *esEngine) getAttributes(m *model.Message, parseResult *message.ParseResult) (*attributes, error) {
	attr := &attributes{}

	attr.To = append(parseResult.Mentions, parseResult.GroupMentions...)
	attr.Citation = parseResult.Citation
	attr.HasURL = strings.Contains(m.Text, "http://") || strings.Contains(m.Text, "https://")
	attr.HasAttachments = len(parseResult.Attachments) != 0

	re, err := regexp.Compile(`https?://[\w,.?=&#%~/-]+`)
	if err != nil {
		// エラー処理整える
		return nil, fmt.Errorf("failed to compile URL regex: %w", err)
	}
	urls := re.FindAllString(m.Text, -1)

	ogpCache := make([]string, 0, len(urls))
	for _, url := range urls {
		tmp, err := e.repo.GetOgpCache(url)
		if err != nil {
			if err != repository.ErrNotFound {
				// エラー処理整える
				e.l.Warn("failed to get ogp cache", zap.String("url", url), zap.Error(err))
			}
			continue
		}
		if tmp.Valid {
			ogpCache = append(ogpCache, tmp.Content.Title + "\n" + tmp.Content.Description)
		}
	}
	attr.OgpContent = ogpCache

	for _, attachmentID := range parseResult.Attachments {
		meta, err := e.repo.GetFileMeta(attachmentID)
		if err != nil {
			e.l.Warn(err.Error(), zap.Error(err))
			continue
		}
		if strings.HasPrefix(meta.Mime, "image/") {
			attr.HasImage = true
		} else if strings.HasPrefix(meta.Mime, "video/") {
			attr.HasVideo = true
		} else if strings.HasPrefix(meta.Mime, "audio/") {
			attr.HasAudio = true
		}
	}

	return attr,  nil
}

func (e *esEngine) syncLoop(done <-chan struct{}) {
	t := time.NewTicker(syncInterval)
	defer t.Stop()
loop:
	for {
		err := e.sync()
		if err != nil {
			e.l.Error(err.Error(), zap.Error(err))
		}

		select {
		case <-t.C:
		case <-done:
			break loop
		}
	}
}

func (e *esEngine) newUserCache() (userCache, error) {
	users, err := e.repo.GetUsers(repository.UsersQuery{})
	if err != nil {
		return nil, err
	}
	e.l.Debug("making user cache of size", zap.Int("size", len(users)))

	cache := make(map[uuid.UUID]bool, len(users))
	for _, u := range users {
		cache[u.GetID()] = u.IsBot()
	}
	return cache, nil
}

// sync メッセージを repository.MessageRepository から読み取り、esへindexします
func (e *esEngine) sync() error {
	e.l.Debug("syncing messages with es")

	lastSynced, err := e.lastInsertedUpdated()
	if err != nil {
		return err
	}

	var userCache userCache
	lastInsert := lastSynced
	for {
		messages, more, err := e.repo.GetUpdatedMessagesAfter(lastInsert, syncMessageBulk)
		if err != nil {
			return err
		}
		if len(messages) == 0 {
			break
		}
		lastInsert = messages[len(messages)-1].UpdatedAt

		// NOTE: index時にBotかどうかを確認するN+1問題へのworkaround
		// ユーザーキャッシュサービスができたら書き換えても良い
		if userCache == nil && more {
			// 新規メッセージが2ページ以上の時のみデータが入ったキャッシュを作成
			userCache, err = e.newUserCache()
			if err != nil {
				return err
			}
		}

		err = syncNewMessages(e, messages, lastInsert, lastSynced, userCache)
		if err != nil {
			return err
		}

		if !more {
			break
		}
	}

	lastDelete := lastSynced
	for {
		messages, more, err := e.repo.GetDeletedMessagesAfter(lastDelete, syncMessageBulk)
		if err != nil {
			return err
		}
		if len(messages) == 0 {
			break
		}
		if !messages[len(messages)-1].DeletedAt.Valid {
			return errors.New("expected DeletedAt to exist, but found nil")
		}
		lastDelete = messages[len(messages)-1].DeletedAt.Time

		err = syncDeletedMessages(e, messages, lastDelete, lastSynced)
		if err != nil {
			return err
		}

		if !more {
			break
		}
	}

	return nil
}

func syncNewMessages(e *esEngine, messages []*model.Message, lastInsert time.Time, lastSynced time.Time, userCache userCache) (err error) {
	bulkIndexer, err := esutil.NewBulkIndexer(esutil.BulkIndexerConfig{
		Client: e.client,
		Index:  getIndexName(esMessageIndex),
	})
	if err != nil {
		return err
	}

	defer func() {
		closeErr := bulkIndexer.Close(context.Background())
		if err != nil && closeErr != nil { // エラーが発生してからdeferに来た時、エラーの上書きを防ぐ。
			err = fmt.Errorf("error in bulk index: %w.\nerror in closing bulk indexer: %w", err, closeErr)
			return
		}
		if closeErr != nil {
			err = closeErr
			return
		}

		e.l.Info(fmt.Sprintf("indexed %v message(s) to index, updated %v message(s) on index, failed %v message(s), last insert %v",
			bulkIndexer.Stats().NumIndexed, bulkIndexer.Stats().NumUpdated, bulkIndexer.Stats().NumFailed, lastInsert))
	}()

	for _, v := range messages {
		if v.CreatedAt.After(lastSynced) {
			doc, err := e.convertMessageCreated(v, message.Parse(v.Text), userCache)
			if err != nil {
				return err
			}

			data, err := json.Marshal(*doc)
			if err != nil {
				return err
			}

			err = bulkIndexer.Add(context.Background(), esutil.BulkIndexerItem{
				Action:     "index",
				DocumentID: v.ID.String(),
				Body:       bytes.NewReader(data),
			})
			if err != nil {
				return err
			}
		} else {
			doc, err := e.convertMessageUpdated(v, message.Parse(v.Text))
			if err != nil {
				return err
			}

			data, err := json.Marshal(map[string]any{"doc": *doc})
			if err != nil {
				return err
			}

			err = bulkIndexer.Add(context.Background(), esutil.BulkIndexerItem{
				Action:     "update",
				DocumentID: v.ID.String(),
				Body:       bytes.NewReader(data),
			})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func syncDeletedMessages(e *esEngine, messages []*model.Message, lastDelete time.Time, lastSynced time.Time) (err error) {
	bulkIndexer, err := esutil.NewBulkIndexer(esutil.BulkIndexerConfig{
		Client: e.client,
		Index:  getIndexName(esMessageIndex),
	})
	if err != nil {
		return err
	}

	defer func() {
		closeErr := bulkIndexer.Close(context.Background())
		if err != nil && closeErr != nil { // エラーが発生してからdeferに来た時、エラーの上書きを防ぐ
			err = fmt.Errorf("error in bulk index: %w.\nerror in closing bulk indexer: %w", err, closeErr)
			return
		}
		if closeErr != nil {
			err = closeErr
			return
		}

		e.l.Info(fmt.Sprintf("deleted %v message(s) from index, failed %v message(s),  last delete %v",
			bulkIndexer.Stats().NumDeleted, bulkIndexer.Stats().NumFailed, lastDelete))
	}()

	for _, v := range messages {
		if v.CreatedAt.After(lastSynced) {
			continue
		}
		err = bulkIndexer.Add(context.Background(), esutil.BulkIndexerItem{
			Action:     "delete",
			DocumentID: v.ID.String(),
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// lastInsertedUpdated esに存在している、updatedAtが一番新しいメッセージの値を取得します
func (e *esEngine) lastInsertedUpdated() (time.Time, error) {
	sr, err := e.client.Search(
		e.client.Search.WithIndex(getIndexName(esMessageIndex)),
		e.client.Search.WithSort("updatedAt:desc"),
		e.client.Search.WithSize(1))
	if err != nil {
		return time.Time{}, err
	}
	defer sr.Body.Close()

	var res esSearchResponse
	err = json.NewDecoder(sr.Body).Decode(&res)
	if err != nil {
		return time.Time{}, err
	}

	lastUpdatedDoc := res.Hits.Hits

	if len(lastUpdatedDoc) == 0 {
		return time.Time{}, nil
	}

	return lastUpdatedDoc[0].Source.UpdatedAt, nil
}
