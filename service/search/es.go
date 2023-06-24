package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/gofrs/uuid"
	"github.com/olivere/elastic/v7"
	"go.uber.org/zap"

	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/service/channel"
	"github.com/traPtitech/traQ/service/message"
)

const (
	esRequiredVersionPrefix = "7."
	esIndexPrefix           = "traq_"
	esMessageIndex          = "message"
	esDateFormat            = "2006-01-02T15:04:05.000000000Z"
)

func getIndexName(index string) string {
	return esIndexPrefix + index
}

// ESEngineConfig Elasticsearch検索エンジン設定
type ESEngineConfig struct {
	// URL ESのURL
	URL []string
}

// esEngine search.Engine 実装
type esEngine struct {
	client *elasticsearch.Client
	mm     message.Manager
	cm     channel.Manager
	repo   repository.Repository
	l      *zap.Logger
	done   chan<- struct{}
}

// esMessageDoc Elasticsearchに入るメッセージの情報
type esMessageDoc struct {
	ID             uuid.UUID   `json:"-"`
	UserID         uuid.UUID   `json:"userId"`
	ChannelID      uuid.UUID   `json:"channelId"`
	IsPublic       bool        `json:"isPublic"`
	Bot            bool        `json:"bot"`
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

type esCreateIndexBody struct {
	Mappings m `json:"mappings"`
	Settings m `json:"settings"`
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
		"isPublic": m{
			"type": "boolean",
		},
		"bot": m{
			"type": "boolean",
		},
		"text": m{
			"type":     "text",
			"analyzer": "sudachi_analyzer",
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

// esSetting Indexに追加するsetting情報
var esSetting = m{
	"index": m{
		"analysis": m{
			"tokenizer": m{
				"sudachi_tokenizer": m{
					"type": "sudachi_tokenizer",
				},
			},
			"filter": m{
				"sudachi_split_filter": m{
					"type": "sudachi_split",
					"mode": "search",
				},
			},
			"analyzer": m{
				"sudachi_analyzer": m{
					"tokenizer": "sudachi_tokenizer",
					"type":      "custom",
					"filter": []string{
						"sudachi_split_filter",
						"sudachi_normalizedform",
					},
					"discard_punctuation": true,
					"resources_path":      "/usr/share/elasticsearch/plugins/analysis-sudachi/",
					"settings_path":       "/usr/share/elasticsearch/plugins/analysis-sudachi/sudachi.json",
				},
			},
		},
	},
}

// NewESEngine Elasticsearch検索エンジンを生成します
func NewESEngine(mm message.Manager, cm channel.Manager, repo repository.Repository, logger *zap.Logger, config ESEngineConfig) (Engine, error) {
	// esクライアント作成
	client, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: config.URL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to init search engine: %w", err)
	}

	// esバージョン確認
	infoRes, err := client.Info()
	if err != nil {
		return nil, fmt.Errorf("failed to get search engine info: %w", err)
	}
	if infoRes.IsError() {
		return nil, fmt.Errorf("failed to get search engine info: %s", infoRes.String())
	}
	defer infoRes.Body.Close()

	var r map[string]interface{}
	if err := json.NewDecoder(infoRes.Body).Decode(&r); err != nil {
		return nil, fmt.Errorf("failed to decode search engine info: %w", err)
	}

	version, ok := r["version"].(map[string]interface{})["number"].(string)
	if !ok {
		return nil, fmt.Errorf("failed to convert version value '%v' to string", r["version"].(map[string]interface{})["number"])
	}
	logger.Info(fmt.Sprintf("Using elasticsearch version %s", version))
	if !strings.HasPrefix(version, esRequiredVersionPrefix) {
		return nil, fmt.Errorf("failed to init search engine: unsupported version (%s). expected major version %s", version, esRequiredVersionPrefix)
	}

	// index確認
	existsRes, err := client.Indices.Exists([]string{getIndexName(esMessageIndex)})
	if err != nil || existsRes.IsError() {
		return nil, fmt.Errorf("failed to init search engine: %w", err)
	}
	if existsRes.StatusCode == http.StatusNotFound {
		body, err := json.Marshal(esCreateIndexBody{
			Mappings: esMapping,
			Settings: esSetting,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to init search engine: %w", err)
		}
		createIndexRes, err := client.Index(getIndexName(esMessageIndex), bytes.NewBuffer(body), client.Index.WithContext(context.Background()))
		if err != nil {
			return nil, fmt.Errorf("failed to init search engine: %w", err)
		}
		defer createIndexRes.Body.Close()
		if err := json.NewDecoder(createIndexRes.Body).Decode(&r); err != nil {
			return nil, fmt.Errorf("failed to decode create index response: %w", err)
		}
		acknowledged, ok := r["acknowledged"].(bool)
		if !ok {
			return nil, fmt.Errorf("failed to convert es index acknowledged value: %v", createIndexRes.String())
		}
		if !acknowledged {
			return nil, fmt.Errorf("failed to create index")
		}
	}

	done := make(chan struct{})
	engine := &esEngine{
		client: client,
		mm:     mm,
		cm:     cm,
		repo:   repo,
		l:      logger.Named("search"),
		done:   done,
	}

	go engine.syncLoop(done)

	return engine, nil
}

func (e *esEngine) Do(q *Query) (Result, error) {
	e.l.Debug("do search", zap.Reflect("q", q))

	var musts []elastic.Query

	if q.Word.Valid {
		musts = append(musts, elastic.NewSimpleQueryStringQuery(q.Word.V).
			Field("text").
			DefaultOperator("AND"))
	}

	switch {
	case q.After.Valid && q.Before.Valid:
		musts = append(musts, elastic.NewRangeQuery("createdAt").
			Gt(q.After.ValueOrZero().Format(esDateFormat)).
			Lt(q.Before.ValueOrZero().Format(esDateFormat)))
	case q.After.Valid && !q.Before.Valid:
		musts = append(musts, elastic.NewRangeQuery("createdAt").
			Gt(q.After.ValueOrZero().Format(esDateFormat)))
	case !q.After.Valid && q.Before.Valid:
		musts = append(musts, elastic.NewRangeQuery("createdAt").
			Lt(q.Before.ValueOrZero().Format(esDateFormat)))
	}

	// チャンネル指定があるときはそのチャンネルを検索
	// そうでないときはPublicチャンネルを検索
	if q.In.Valid {
		musts = append(musts, elastic.NewTermQuery("channelId", q.In))
	} else {
		musts = append(musts, elastic.NewTermQuery("isPublic", true))
	}

	if q.To.Valid {
		musts = append(musts, elastic.NewTermQuery("to", q.To))
	}

	if q.From.Valid {
		musts = append(musts, elastic.NewTermQuery("userId", q.From))
	}

	if q.Citation.Valid {
		musts = append(musts, elastic.NewTermQuery("citation", q.Citation))
	}

	if q.Bot.Valid {
		musts = append(musts, elastic.NewTermQuery("bot", q.Bot))
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

	limit, offset := 20, 0
	if q.Limit.Valid {
		limit = q.Limit.V
	}
	if q.Offset.Valid {
		offset = q.Offset.V
	}

	// NOTE: 現状`sort.Key`はそのままesのソートキーとして使える前提
	sort := q.GetSortKey()



	sr, err := e.client.Search().
		Index(getIndexName(esMessageIndex)).
		Query(elastic.NewBoolQuery().Must(musts...)).
		Sort(sort.Key, !sort.Desc).
		Size(limit).
		From(offset).
		Do(context.Background())
	if err != nil {
		return nil, err
	}

	e.l.Debug("search result", zap.Reflect("hits", sr.Hits))
	return e.bindESResult(sr)
}

func (e *esEngine) Available() bool {
	return e.client.IsRunning()
}

func (e *esEngine) Close() error {
	e.client.Stop()
	e.done <- struct{}{}
	return nil
}
