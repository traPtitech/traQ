package search

import (
	"context"
	"errors"
	"fmt"
	"github.com/gofrs/uuid"
	json "github.com/json-iterator/go"
	"github.com/olivere/elastic/v7"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils/message"
	"go.uber.org/zap"
	"strings"
	"time"
)

const (
	syncInterval    = 1 * time.Minute
	syncMessageBulk = 100
)

type attributes struct {
	To             []uuid.UUID
	Citation       []uuid.UUID
	HasURL         bool
	HasAttachments bool
	HasImage       bool
	HasVideo       bool
	HasAudio       bool
}

// convertMessageCreated 新規メッセージをesへ入れる型に変換する
func (e *esEngine) convertMessageCreated(m *model.Message, parseResult *message.ParseResult) *esMessageDoc {
	attr := e.getAttributes(m, parseResult)
	return &esMessageDoc{
		UserID:         m.UserID,
		ChannelID:      m.ChannelID,
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
	}
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
		meta, err := e.repo.GetFileMeta(attachmentID)
		if err != nil {
			e.l.Error(err.Error(), zap.Error(err))
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
		select {
		case <-t.C:
			err := e.sync()
			if err != nil {
				e.l.Error(err.Error(), zap.Error(err))
			}
		case <-done:
			break loop
		}
	}
}

// sync メッセージを repository.MessageRepository から読み取り、esへindexします
func (e *esEngine) sync() error {
	e.l.Debug("syncing messages with es")

	lastSynced, err := e.lastInsertedUpdated()
	if err != nil {
		return err
	}

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

		bulk := e.client.Bulk().Index(getIndexName(esMessageIndex))
		for _, v := range messages {
			var bulkReq elastic.BulkableRequest
			if v.CreatedAt.After(lastSynced) {
				bulkReq = elastic.NewBulkIndexRequest().
					Id(v.ID.String()).
					Doc(e.convertMessageCreated(v, message.Parse(v.Text)))
			} else {
				bulkReq = elastic.NewBulkUpdateRequest().
					Id(v.ID.String()).
					Doc(e.convertMessageUpdated(v, message.Parse(v.Text)))
			}
			bulk.Add(bulkReq)
		}
		res, err := bulk.Do(context.Background())
		if err != nil {
			return err
		}

		e.l.Debug(fmt.Sprintf("indexed %v, updated %v message(s)", len(res.Indexed()), len(res.Updated())))

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
		if messages[len(messages)-1].DeletedAt == nil {
			return errors.New("expected DeletedAt to exist, but found nil")
		}
		lastDelete = *messages[len(messages)-1].DeletedAt

		bulk := e.client.Bulk().Index(getIndexName(esMessageIndex))
		count := 0
		for _, v := range messages {
			if v.CreatedAt.After(lastSynced) {
				continue
			}
			count++
			bulk.Add(
				elastic.NewBulkDeleteRequest().
					Id(v.ID.String()),
			)
		}
		if count == 0 {
			if more {
				continue
			} else {
				break
			}
		}
		res, err := bulk.Do(context.Background())
		if err != nil {
			return err
		}

		e.l.Debug(fmt.Sprintf("deleted %v message(s)", len(res.Deleted())))

		if !more {
			break
		}
	}

	return nil
}

// lastInsertedUpdated esに存在している、updatedAtが一番新しいメッセージの値を取得します
func (e *esEngine) lastInsertedUpdated() (time.Time, error) {
	sr, err := e.client.Search().
		Index(getIndexName(esMessageIndex)).
		Sort("updatedAt", false).
		Size(1).
		Do(context.Background())

	if err != nil {
		return time.Time{}, err
	}
	if len(sr.Hits.Hits) == 0 {
		return time.Time{}, nil
	}

	var m esMessageDoc
	hit := sr.Hits.Hits[0]
	if err := json.Unmarshal(hit.Source, &m); err != nil {
		return time.Time{}, err
	}
	return m.UpdatedAt, nil
}
