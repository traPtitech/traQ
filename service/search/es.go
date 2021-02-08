package search

import (
	"context"
	"fmt"
	"github.com/gofrs/uuid"
	json "github.com/json-iterator/go"
	"github.com/leandro-lugaresi/hub"
	"github.com/olivere/elastic/v7"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/utils/message"
	"go.uber.org/zap"
	"strings"
	"time"
)

const (
	esRequiredVersion = "7.8.0"
	esIndexPrefix     = "traq_"
	esMessageIndex    = "message"
)

// ESEngineConfig Elasticsearch検索エンジン設定
type ESEngineConfig struct {
	// URL ESのURL
	URL string
}

// esEngine search.Engine 実装
type esEngine struct {
	client *elastic.Client
	hub    *hub.Hub
	repo   repository.Repository
	l      *zap.Logger
}

// esMessageDoc Elasticsearchに入るメッセージの情報
type esMessageDoc struct {
	ID             uuid.UUID   `json:"-"`
	UserID         uuid.UUID   `json:"userId"`
	ChannelID      uuid.UUID   `json:"channelId"`
	Text           string      `json:"text"`
	CreatedAt      time.Time   `json:"createdAt"`
	UpdatedAt      time.Time   `json:"updatedAt"`
	To             []uuid.UUID `json:"to"`
	Citation       []uuid.UUID `json:"citation"`
	HasURL         bool        `json:"hasURL"`
	HasAttachments bool        `json:"hasAttachments"`
	HasImage       bool        `json:"hasImage"`
	HasVideo       bool        `json:"hasVideo"`
	HasAudio       bool        `json:"hasAudio"`
}

// esMessageDocUpdate Update用 Elasticsearchに入るメッセージの部分的な情報
type esMessageDocUpdate struct {
	Text           string      `json:"text"`
	UpdatedAt      time.Time   `json:"updatedAt"`
	Citation       []uuid.UUID `json:"citation"`
	HasURL         bool        `json:"hasURL"`
	HasAttachments bool        `json:"hasAttachments"`
	HasImage       bool        `json:"hasImage"`
	HasVideo       bool        `json:"hasVideo"`
	HasAudio       bool        `json:"hasAudio"`
}

type m map[string]interface{}

// esMapping Elasticsearchに入るメッセージの情報
// esMessageDoc と同じにする
var esMapping = m{
	"properties": m{
		"userId": m{
			"type": "keyword",
		},
		"channelId": m{
			"type": "keyword",
		},
		"text": m{
			"type": "text",
		},
		"createdAt": m{
			"type":   "date",
			"format": "strict_date_optional_time_nanos", // 2006-01-02T15:04:05.7891011Z
		},
		"updatedAt": m{
			"type":   "date",
			"format": "strict_date_optional_time_nanos",
		},
		"to": m{
			"type": "keyword",
		},
		"citation": m{
			"type": "keyword",
		},
		"hasURL": m{
			"type": "boolean",
		},
		"hasAttachments": m{
			"type": "boolean",
		},
		"hasImage": m{
			"type": "boolean",
		},
		"hasVideo": m{
			"type": "boolean",
		},
		"hasAudio": m{
			"type": "boolean",
		},
	},
}

// esResult search.Result 実装
type esResult struct {
	docs []*esMessageDoc
}

type attributes struct {
	To             []uuid.UUID
	Citation       []uuid.UUID
	HasURL         bool
	HasAttachments bool
	HasImage       bool
	HasVideo       bool
	HasAudio       bool
}

func (e *esResult) Get() map[uuid.UUID]string {
	r := make(map[uuid.UUID]string, len(e.docs))
	for _, doc := range e.docs {
		r[doc.ID] = doc.Text
	}
	return r
}

// NewESEngine Elasticsearch検索エンジンを生成します
func NewESEngine(hub *hub.Hub, repo repository.Repository, logger *zap.Logger, config ESEngineConfig) (Engine, error) {
	// es接続
	client, err := elastic.NewClient(elastic.SetURL(config.URL))
	if err != nil {
		return nil, fmt.Errorf("failed to init search engine: %w", err)
	}

	// esバージョン確認
	version, err := client.ElasticsearchVersion(config.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch es version: %w", err)
	}
	if esRequiredVersion != version {
		return nil, fmt.Errorf("failed to init search engine: version mismatch (%s)", version)
	}

	// index確認
	if exists, err := client.IndexExists(getIndexName(esMessageIndex)).Do(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to init search engine: %w", err)
	} else if !exists {
		// index作成
		r1, err := client.CreateIndex(getIndexName(esMessageIndex)).Do(context.Background())
		if err != nil {
			return nil, fmt.Errorf("failed to init search engine: %w", err)
		}
		if !r1.Acknowledged {
			return nil, fmt.Errorf("failed to init search engine: index not acknowledged")
		}

		// mapping作成
		r2, err := client.PutMapping().Index(getIndexName(esMessageIndex)).BodyJson(esMapping).Do(context.Background())
		if err != nil {
			return nil, fmt.Errorf("failed to init search engine: %w", err)
		}
		if !r2.Acknowledged {
			return nil, fmt.Errorf("failed to init search engine: mapping not acknowledged")
		}
	}

	engine := &esEngine{
		client: client,
		hub:    hub,
		repo:   repo,
		l:      logger.Named("search"),
	}

	go engine.syncLoop()

	return engine, nil
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

func (e *esEngine) Do(q *Query) (Result, error) {
	// TODO 実装
	e.l.Debug("do search", zap.Reflect("q", q))

	// TODO "should" "must not"をどういれるか
	var musts []elastic.Query

	// TODO MatchQuery, MatchPhraseQuery(語順が重要な場合)との出し分け
	if q.Word.Valid {
		musts = append(musts, elastic.NewMatchPhraseQuery("text", q.Word))
	}

	switch {
	case q.After.Valid && q.Before.Valid:
		musts = append(musts, elastic.NewRangeQuery("createdAt").Gte(q.After.ValueOrZero().Format("2006-01-02T15:04:05Z")).Lte(q.Before.ValueOrZero().Format("2006-01-02T15:04:05Z")))
	case q.After.Valid && !q.Before.Valid:
		musts = append(musts, elastic.NewRangeQuery("date").Gte(q.After.ValueOrZero().Format("2006-01-02 15:04:05Z")))
		fmt.Println(elastic.NewRangeQuery("date").Gte(q.After.ValueOrZero().Format("2006-01-02T15:04:05Z")).Format("strict_date_time_no_millis"))
	case !q.After.Valid && q.Before.Valid:
		musts = append(musts, elastic.NewRangeQuery("createdAt").Lte(q.Before.ValueOrZero().Format("2006-01-02T15:04:05Z")))
	}

	switch {
	case q.To.Valid:
		musts = append(musts, elastic.NewTermQuery("to", q.To))
	case q.From.Valid:
		musts = append(musts, elastic.NewTermQuery("userId", q.From))
	}

	if q.Citation.Valid {
		musts = append(musts, elastic.NewTermQuery("citation", q.Citation))
	}

	if q.HasURL.Valid {
		musts = append(musts, elastic.NewTermQuery("hasURL", q.HasURL))
	}

	if q.HasAttachments.Valid {
		musts = append(musts, elastic.NewTermQuery("hasAttachments", q.HasAttachments))
	}

	if q.HasImage.Valid {
		musts = append(musts, elastic.NewTermQuery("hasImage", q.HasImage))
	}
	if q.HasVideo.Valid {
		musts = append(musts, elastic.NewTermQuery("hasVideo", q.HasVideo))
	}
	if q.HasAudio.Valid {
		musts = append(musts, elastic.NewTermQuery("hasAudio", q.HasAudio))
	}

	sr, err := e.client.Search().
		Index(getIndexName(esMessageIndex)).
		Query(elastic.NewBoolQuery().Must(musts...)).
		Sort("createdAt", false).
		Size(20).
		Do(context.Background())
	if err != nil {
		return nil, err
	}

	r := &esResult{}
	e.l.Debug("search result", zap.Reflect("hits", sr.Hits))
	for _, hit := range sr.Hits.Hits {
		var m esMessageDoc
		if err := json.Unmarshal(hit.Source, &m); err != nil {
			return nil, err
		}
		m.ID = uuid.Must(uuid.FromString(hit.Id))
		r.docs = append(r.docs, &m)
	}

	return r, nil
}

func (e *esEngine) Available() bool {
	return e.client.IsRunning()
}

func (e *esEngine) Close() error {
	e.client.Stop()
	return nil
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

func getIndexName(index string) string {
	return esIndexPrefix + index
}
