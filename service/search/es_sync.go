package search

import (
	"bytes"
	"context"
	"errors"
	"fmt"
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
	HasURL         bool
	HasAttachments bool
	HasImage       bool
	HasVideo       bool
	HasAudio       bool
	// ImageFileIDs アニメーションでない画像の添付ファイルIDリスト（非同期画像処理用）
	ImageFileIDs []uuid.UUID
}

// ユーザーがbotかどうかのcache
type userCache map[uuid.UUID]bool

// convertMessageCreated 新規メッセージをesへ入れる型に変換する
func (e *esEngine) convertMessageCreated(m *model.Message, parseResult *message.ParseResult, userCache userCache) (*esMessageDoc, error) {
	var isBot, ok bool
	if isBot, ok = userCache[m.UserID]; !ok {
		// 新規ユーザー or キャッシュが存在しない
		user, err := e.repo.GetUser(context.Background(), m.UserID, false)
		if err != nil {
			return nil, err
		}
		isBot = user.IsBot()
	}

	attr := e.getAttributes(m, parseResult)

	return &esMessageDoc{
		UserID:         m.UserID,
		ChannelID:      m.ChannelID,
		IsPublic:       e.cm.IsPublicChannel(context.Background(), m.ChannelID),
		Bot:            isBot,
		Text:           m.Text,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
		To:             attr.To,
		Citation:       attr.Citation,
		HasURL:         attr.HasURL,
		HasAttachments: attr.HasAttachments,
		HasImage:       attr.HasImage,
		HasVideo:       attr.HasVideo,
		HasAudio:       attr.HasAudio,
	}, nil
}

// convertMessageUpdated 既存メッセージの更新情報をesへ入れる型に変換する
func (e *esEngine) convertMessageUpdated(m *model.Message, parseResult *message.ParseResult) *esMessageDocUpdate {
	attr := e.getAttributes(m, parseResult)
	// Updateする項目のみ
	return &esMessageDocUpdate{
		Text:           m.Text,
		UpdatedAt:      m.UpdatedAt,
		Citation:       attr.Citation,
		HasURL:         attr.HasURL,
		HasAttachments: attr.HasAttachments,
		HasImage:       attr.HasImage,
		HasVideo:       attr.HasVideo,
		HasAudio:       attr.HasAudio,
	}
}

func (e *esEngine) getAttributes(m *model.Message, parseResult *message.ParseResult) *attributes {
	attr := &attributes{}

	attr.To = append(parseResult.Mentions, parseResult.GroupMentions...)
	attr.Citation = parseResult.Citation
	attr.HasURL = strings.Contains(m.Text, "http://") || strings.Contains(m.Text, "https://")
	attr.HasAttachments = len(parseResult.Attachments) != 0

	for _, attachmentID := range parseResult.Attachments {
		meta, err := e.repo.GetFileMeta(context.Background(), attachmentID)
		if err != nil {
			e.l.Warn(err.Error(), zap.Error(err))
			continue
		}
		if strings.HasPrefix(meta.Mime, "image/") {
			attr.HasImage = true
			// アニメーション画像でない画像のファイルIDを収集
			if !meta.IsAnimatedImage {
				attr.ImageFileIDs = append(attr.ImageFileIDs, attachmentID)
			}
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
	users, err := e.repo.GetUsers(context.Background(), repository.UsersQuery{})
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
		messages, more, err := e.repo.GetUpdatedMessagesAfter(context.Background(), lastInsert, syncMessageBulk)
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
		messages, more, err := e.repo.GetDeletedMessagesAfter(context.Background(), lastDelete, syncMessageBulk)
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

// imageProcessTask 非同期画像処理のタスク
type imageProcessTask struct {
	MessageID    uuid.UUID
	ImageFileIDs []uuid.UUID
}

func syncNewMessages(e *esEngine, messages []*model.Message, lastInsert time.Time, lastSynced time.Time, userCache userCache) (err error) {
	bulkIndexer, err := esutil.NewBulkIndexer(esutil.BulkIndexerConfig{
		Client: e.client,
		Index:  getIndexName(esMessageIndex),
	})
	if err != nil {
		return err
	}

	var imageTasks []imageProcessTask

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

		// 非同期で画像処理を開始
		if len(imageTasks) > 0 && e.imageClient.Available() {
			go e.processImageTasks(imageTasks)
		}
	}()

	for _, v := range messages {
		if v.CreatedAt.After(lastSynced) {
			parseResult := message.Parse(v.Text)
			doc, err := e.convertMessageCreated(v, parseResult, userCache)
			if err != nil {
				return err
			}

			// 画像が含まれている場合、非同期処理タスクを作成
			attr := e.getAttributes(v, parseResult)
			if len(attr.ImageFileIDs) > 0 {
				imageTasks = append(imageTasks, imageProcessTask{
					MessageID:    v.ID,
					ImageFileIDs: attr.ImageFileIDs,
				})
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

	return nil
}

// processImageTasks 画像処理タスクを非同期で実行し、ESインデックスを更新する
func (e *esEngine) processImageTasks(tasks []imageProcessTask) {
	for _, task := range tasks {
		if err := e.processImageTask(task); err != nil {
			e.l.Warn("failed to process image task",
				zap.Stringer("messageID", task.MessageID),
				zap.Error(err))
		}
	}
}

// processImageTask 1メッセージ分の画像処理を実行する
func (e *esEngine) processImageTask(task imageProcessTask) error {
	// 署名付きURLを生成
	imageURLs := make([]string, 0, len(task.ImageFileIDs))
	for _, fileID := range task.ImageFileIDs {
		url, err := e.generateImageURL(fileID)
		if err != nil {
			e.l.Warn("failed to generate image URL",
				zap.Stringer("fileID", fileID),
				zap.Error(err))
			continue
		}
		if url == "" {
			continue
		}
		imageURLs = append(imageURLs, url)
	}
	if len(imageURLs) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), e.imageConfig.Timeout)
	defer cancel()

	results, err := e.imageClient.ProcessImages(ctx, imageURLs)
	if err != nil {
		return fmt.Errorf("failed to process images: %w", err)
	}

	// OCR結果を結合
	var texts []string
	var vectors []esImageVector
	for _, r := range results {
		if r.Text != "" {
			texts = append(texts, r.Text)
		}
		if r.Vector != nil {
			vectors = append(vectors, esImageVector{Vector: r.Vector})
		}
	}

	// ESインデックスを部分更新
	update := esImageDocUpdate{
		ImageText:    strings.Join(texts, "\n"),
		ImageVectors: vectors,
	}

	data, err := json.Marshal(map[string]any{"doc": update})
	if err != nil {
		return fmt.Errorf("failed to marshal image update: %w", err)
	}

	res, err := e.client.Update(
		getIndexName(esMessageIndex),
		task.MessageID.String(),
		bytes.NewReader(data),
	)
	if err != nil {
		return fmt.Errorf("failed to update image data in ES: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("ES update error for message %s: %s", task.MessageID, res.String())
	}

	e.l.Info("updated image data for message",
		zap.Stringer("messageID", task.MessageID),
		zap.Int("imageCount", len(results)))
	return nil
}

// ProcessImagesForMessages バッチ用: 指定メッセージの画像を処理してインデックスを更新する
func (e *esEngine) ProcessImagesForMessages(ctx context.Context, messages []*model.Message) error {
	for _, msg := range messages {
		parseResult := message.Parse(msg.Text)
		attr := e.getAttributes(msg, parseResult)
		if len(attr.ImageFileIDs) == 0 {
			continue
		}

		task := imageProcessTask{
			MessageID:    msg.ID,
			ImageFileIDs: attr.ImageFileIDs,
		}
		if err := e.processImageTask(task); err != nil {
			e.l.Warn("batch: failed to process image task",
				zap.Stringer("messageID", msg.ID),
				zap.Error(err))
			// バッチ処理ではエラーをスキップして続行
			continue
		}

		// コンテキストキャンセルを確認
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
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
