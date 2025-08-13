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
	"github.com/samber/lo"
	"go.uber.org/zap"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	resMessage "github.com/traPtitech/traQ/service/message"
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

	attr := e.getAttributes(m.Text, parseResult)

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
func (e *esEngine) convertMessageUpdated(m *model.Message, parseResult *message.ParseResult) *esMessageDocUpdate {
	attr := e.getAttributes(m.Text, parseResult)
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
	}
}

// convertResMessageUpdated 既存のesMessage.Message型のメッセージの更新情報をesへ入れる型に変換する
func (e *esEngine) convertResMessageUpdated(m resMessage.Message, parseResult *message.ParseResult) *esMessageDocUpdate {
	attr := e.getAttributes(m.GetText(), parseResult)
	// Updateする項目のみ
	return &esMessageDocUpdate{
		Text:           m.GetText(),
		UpdatedAt:      m.GetUpdatedAt(),
		Citation:       attr.Citation,
		OgpContent:     attr.OgpContent,
		HasURL:         attr.HasURL,
		HasAttachments: attr.HasAttachments,
		HasImage:       attr.HasImage,
		HasVideo:       attr.HasVideo,
		HasAudio:       attr.HasAudio,
	}
}

func (e *esEngine) getAttributes(messageText string, parseResult *message.ParseResult) *attributes {
	attr := &attributes{}

	attr.To = append(parseResult.Mentions, parseResult.GroupMentions...)
	attr.Citation = parseResult.Citation
	attr.HasURL = strings.Contains(messageText, "http://") || strings.Contains(messageText, "https://")
	attr.HasAttachments = len(parseResult.Attachments) != 0

	urlRegex := regexp.MustCompile(`(^https?|[^a-zA-Z0-9:+]https?)://[^\s\(\)\{\}\[\]]+`)
	urls := lo.Map(urlRegex.FindAllString(messageText, -1), func(url string, _ int) string {
		if url[0] != 'h' {
			return url[1:]
		} else {
			return url
		}
	})

	filteredUrls := lo.Filter(urls, func(url string, _ int) bool {
		return !strings.HasPrefix(url, "https://q.trap.jp")
	})

	ogpCache := make([]string, 0, len(filteredUrls))
	for _, url := range filteredUrls {
		urlCache, err := e.repo.GetOgpCache(url)
		if err != nil {
			e.l.Warn(err.Error(), zap.Error(err))
			continue
		}
		if urlCache.Valid {
			ogpCache = append(ogpCache, urlCache.Content.Title + "\n" + urlCache.Content.Description)
		}
	}
	if len(ogpCache) == 0 {
		ogpCache = append(ogpCache, "")
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

	return attr
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
		if len(messages) != 0 {
			lastInsert = messages[len(messages)-1].UpdatedAt
		}
		e.l.Debug("fetched messages", zap.Int("count", len(messages)), zap.Time("lastInsert", lastInsert))


		r, err := e.getNoOgpfieldMessage(syncMessageBulk - len(messages))
		if err != nil {
			return fmt.Errorf("failed to get messages without OGP field: %w", err)
		}
		e.l.Debug("fetched messages without OGP field", zap.Int("count", len(r.Hits())))


		if r.TotalHits() == 0 && len(messages) == 0 {
			break
		}

		// NOTE: index時にBotかどうかを確認するN+1問題へのworkaround
		// ユーザーキャッシュサービスができたら書き換えても良い
		if userCache == nil && more {
			// 新規メッセージが2ページ以上の時のみデータが入ったキャッシュを作成
			userCache, err = e.newUserCache()
			if err != nil {
				return err
			}
		}
		err = syncNewMessages(e, messages, r.Hits(), lastInsert, lastSynced, userCache)
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

func syncNewMessages(e *esEngine, messages []*model.Message, noOgpMessage []resMessage.Message, lastInsert time.Time, lastSynced time.Time, userCache userCache) (err error) {
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
			doc := e.convertMessageUpdated(v, message.Parse(v.Text))

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

	for _, v := range noOgpMessage {
		doc := e.convertResMessageUpdated(v, message.Parse(v.GetText()))

		data, err := json.Marshal(map[string]any{"doc": *doc})
		if err != nil {
			return err
		}

		err = bulkIndexer.Add(context.Background(), esutil.BulkIndexerItem{
			Action:     "update",
			DocumentID: v.GetID().String(),
			Body:       bytes.NewReader(data),
		})
		if err != nil {
			return err
		}
		e.l.Info(fmt.Sprintf("updated noOgpMessage %s", v.GetID().String()), zap.Time("updatedAt", v.GetUpdatedAt()), zap.Time("createdAt", v.GetCreatedAt()))


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

// getNoOgpfieldMessage OGPフィールドを持っていない過去のメッセージを取得する
func (e *esEngine) getNoOgpfieldMessage(limit int) (Result, error) {

	e.l.Debug("getting messages without OGP content", zap.Int("limit", limit))
	type fieldQuery struct {
		Field string `json:"field"`
	}
	
	var mustNots = []searchQuery{
		searchQuery{"exists": fieldQuery{Field: "ogpContent"}},
	}
	var musts = []searchQuery{
		searchQuery{"term": termQuery{"hasURL": termQueryParameter{Value: true}}},
	}
	
	body := newSearchBodyWithMustNot(musts, mustNots)

	e.l.Debug("searching for messages without OGP content", zap.Reflect("body", body))

	b, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal search query: %w", err)
	}
	// OGP情報がないメッセージを取得する
	sr, err := e.client.Search(
		e.client.Search.WithIndex(getIndexName(esMessageIndex)),
		e.client.Search.WithBody(bytes.NewBuffer(b)),
		e.client.Search.WithSort("updatedAt:desc"),
		e.client.Search.WithSize(limit), // 一度に取得する件数
	)
	if err != nil {
		return nil, err
	}
	if sr.IsError() {
		return nil, fmt.Errorf("error in search: %s", sr.String())
	}
	defer sr.Body.Close()

	var res esSearchResponse
	err = json.NewDecoder(sr.Body).Decode(&res)
	if err != nil {
		return nil, err
	}

	e.l.Debug("search result", zap.Reflect("hits", res.Hits))
	return e.parseResultFromResponse(res)

}