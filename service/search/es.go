package search

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	json "github.com/json-iterator/go"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/gofrs/uuid"
	"go.uber.org/zap"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/service/channel"
	"github.com/traPtitech/traQ/service/message"
	"github.com/traPtitech/traQ/utils/storage"
)

const (
	esRequiredVersionPrefix = "8."
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
	URL string
	// Username ESのユーザー名
	Username string
	// Password ESのパスワード
	Password string
	// ImageSearch 画像検索設定
	ImageSearch ImageSearchConfig
}

// esEngine search.Engine 実装
type esEngine struct {
	client      *elasticsearch.Client
	mm          message.Manager
	cm          channel.Manager
	repo        repository.Repository
	fs          storage.FileStorage
	imageClient ImageSearchClient
	imageConfig ImageSearchConfig
	l           *zap.Logger
	done        chan<- struct{}
}

// esImageVector 画像ごとのEmbeddingベクトル
type esImageVector struct {
	Vector []float64 `json:"vector"`
}

// esMessageDoc Elasticsearchに入るメッセージの情報
type esMessageDoc struct {
	ID             uuid.UUID       `json:"-"`
	UserID         uuid.UUID       `json:"userId"`
	ChannelID      uuid.UUID       `json:"channelId"`
	IsPublic       bool            `json:"isPublic"`
	Bot            bool            `json:"bot"`
	Text           string          `json:"text"`
	CreatedAt      time.Time       `json:"createdAt"`
	UpdatedAt      time.Time       `json:"updatedAt"`
	To             []uuid.UUID     `json:"to"`
	Citation       []uuid.UUID     `json:"citation"`
	HasURL         bool            `json:"hasURL"`
	HasAttachments bool            `json:"hasAttachments"`
	HasImage       bool            `json:"hasImage"`
	HasVideo       bool            `json:"hasVideo"`
	HasAudio       bool            `json:"hasAudio"`
	ImageText      string          `json:"imageText,omitempty"`
	ImageVectors   []esImageVector `json:"imageVector,omitempty"`
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

// esImageDocUpdate 画像処理結果によるインデックスの部分更新
type esImageDocUpdate struct {
	ImageText    string          `json:"imageText,omitempty"`
	ImageVectors []esImageVector `json:"imageVector,omitempty"`
}

type esCreateIndexBody struct {
	Mappings m `json:"mappings"`
	Settings m `json:"settings"`
}

type m map[string]any

// esMessageMapping Elasticsearchに入るメッセージのマッピングを生成する
// vectorDimension が 0 の場合は画像ベクトル関連フィールドを含めない
func esMessageMapping(vectorDimension int) m {
	properties := m{
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
		"imageText": m{
			"type":     "text",
			"analyzer": "sudachi_analyzer",
		},
	}
	if vectorDimension > 0 {
		properties["imageVector"] = m{
			"type": "nested",
			"properties": m{
				"vector": m{
					"type": "dense_vector",
					"dims": vectorDimension,
				},
			},
		}
	}
	return m{"properties": properties}
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
func NewESEngine(mm message.Manager, cm channel.Manager, repo repository.Repository, fs storage.FileStorage, logger *zap.Logger, config ESEngineConfig) (Engine, error) {
	// esクライアント作成
	client, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{config.URL},
		Username:  config.Username,
		Password:  config.Password,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to init Elasticsearch: %w", err)
	}

	// esバージョン確認
	infoRes, err := client.Info()
	if err != nil {
		return nil, fmt.Errorf("failed to get Elasticsearch info: %w", err)
	}
	if infoRes.IsError() {
		return nil, fmt.Errorf("failed to get Elasticsearch info: %s", infoRes.String())
	}
	defer infoRes.Body.Close()

	var r struct {
		Version struct {
			Number string `json:"number"`
		} `json:"version"`
	}
	err = json.NewDecoder(infoRes.Body).Decode(&r)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal Elasticsearch info: %w", err)
	}

	if !strings.HasPrefix(r.Version.Number, esRequiredVersionPrefix) {
		return nil, fmt.Errorf("failed to init Elasticsearch: unsupported version (%s). expected major version %s", r.Version.Number, esRequiredVersionPrefix)
	}
	logger.Info(fmt.Sprintf("Using elasticsearch version %s", r.Version.Number))

	// index確認
	existsRes, err := client.Indices.Exists([]string{getIndexName(esMessageIndex)})
	if err != nil {
		return nil, fmt.Errorf("failed to check index exists: %w", err)
	}
	if existsRes.IsError() && existsRes.StatusCode != http.StatusNotFound {
		return nil, fmt.Errorf("failed to check index exists: %s", existsRes.String())
	}
	defer existsRes.Body.Close()

	// indexが存在しなかったら作成
	if existsRes.StatusCode == http.StatusNotFound {
		reqBody, err := json.Marshal(esCreateIndexBody{
			Mappings: esMessageMapping(config.ImageSearch.VectorDimension),
			Settings: esSetting,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to init Elasticsearch: %w", err)
		}

		createIndexRes, err := client.Indices.Create(
			getIndexName(esMessageIndex),
			client.Indices.Create.WithBody(bytes.NewBuffer(reqBody)),
			client.Indices.Create.WithContext(context.Background()))
		if err != nil {
			return nil, fmt.Errorf("failed to create Elasticsearch index: %w", err)
		}
		if createIndexRes.IsError() || createIndexRes.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("failed to create Elasticsearch index: StatusCode: %v, ResponseBody: %v", createIndexRes.StatusCode, createIndexRes.Body)
		}
		defer createIndexRes.Body.Close()
	}

	imageClient := NewImageSearchClient(config.ImageSearch, logger)

	done := make(chan struct{})
	engine := &esEngine{
		client:      client,
		mm:          mm,
		cm:          cm,
		repo:        repo,
		fs:          fs,
		imageClient: imageClient,
		imageConfig: config.ImageSearch,
		l:           logger.Named("search"),
		done:        done,
	}

	go engine.syncLoop(done)

	return engine, nil
}

type searchQuery m

type searchBody struct {
	Query searchQuery `json:"query,omitempty"`
}

func newSearchBody(andQueries []searchQuery) searchBody {
	return searchBody{
		Query: searchQuery{
			"bool": boolQuery{
				Must: andQueries,
			},
		},
	}
}

type simpleQueryString struct {
	Query           string   `json:"query"`
	Fields          []string `json:"fields"`
	DefaultOperator string   `json:"default_operator"`
}

type boolQuery struct {
	Must   []searchQuery `json:"must,omitempty"`
	Should []searchQuery `json:"should,omitempty"`
}

type rangeQuery map[string]rangeParameters

type rangeParameters struct {
	Lt string `json:"lt,omitempty"`
	Gt string `json:"gt,omitempty"`
}

type termQuery map[string]termQueryParameter

type termQueryParameter struct {
	Value any `json:"value,omitempty"`
}

func (e *esEngine) Do(q *Query) (Result, error) {
	e.l.Debug("do search", zap.Reflect("q", q))

	var musts []searchQuery

	if q.Word.Valid {
		body := simpleQueryString{
			Query:           q.Word.V,
			Fields:          []string{"text", "imageText"},
			DefaultOperator: "AND",
		}

		musts = append(musts, searchQuery{"simple_query_string": body})
	}

	switch {
	case q.After.Valid && q.Before.Valid:
		musts = append(musts, searchQuery{"range": rangeQuery{"createdAt": rangeParameters{
			Gt: q.After.ValueOrZero().Format(esDateFormat),
			Lt: q.Before.ValueOrZero().Format(esDateFormat),
		}}})
	case q.After.Valid && !q.Before.Valid:
		musts = append(musts, searchQuery{"range": rangeQuery{"createdAt": rangeParameters{
			Gt: q.After.ValueOrZero().Format(esDateFormat),
		}}})
	case !q.After.Valid && q.Before.Valid:
		musts = append(musts, searchQuery{"range": rangeQuery{"createdAt": rangeParameters{
			Lt: q.Before.ValueOrZero().Format(esDateFormat),
		}}})
	}

	// チャンネル指定があるときはそのチャンネルを検索
	// そうでないときはPublicチャンネルを検索
	if q.In.Valid {
		musts = append(musts, searchQuery{"term": termQuery{"channelId": termQueryParameter{Value: q.In}}})
	} else {
		musts = append(musts, searchQuery{"term": termQuery{"isPublic": termQueryParameter{Value: true}}})
	}

	if len(q.To) > 0 {
		orQueries := make([]searchQuery, 0, len(q.To))
		for _, toID := range q.To {
			orQueries = append(orQueries, searchQuery{"term": termQuery{"to": termQueryParameter{Value: toID}}})
		}

		sq := searchQuery{"bool": boolQuery{Should: orQueries}}
		if len(q.To) == 1 {
			// OR検索が不要
			sq = orQueries[0]
		}

		musts = append(musts, sq)
	}

	if len(q.From) > 0 {
		orQueries := make([]searchQuery, 0, len(q.From))
		for _, fromID := range q.From {
			orQueries = append(orQueries, searchQuery{"term": termQuery{"userId": termQueryParameter{Value: fromID}}})
		}

		sq := searchQuery{"bool": boolQuery{Should: orQueries}}
		if len(q.From) == 1 {
			// OR検索が不要
			sq = orQueries[0]
		}

		musts = append(musts, sq)
	}

	if q.Citation.Valid {
		musts = append(musts, searchQuery{"term": termQuery{"citation": termQueryParameter{Value: q.Citation}}})
	}

	if q.Bot.Valid {
		musts = append(musts, searchQuery{"term": termQuery{"bot": termQueryParameter{Value: q.Bot}}})
	}

	if q.HasURL.Valid {
		musts = append(musts, searchQuery{"term": termQuery{"hasURL": termQueryParameter{Value: q.HasURL}}})
	}

	if q.HasAttachments.Valid {
		musts = append(musts, searchQuery{"term": termQuery{"hasAttachments": termQueryParameter{Value: q.HasAttachments}}})
	}

	if q.HasImage.Valid {
		musts = append(musts, searchQuery{"term": termQuery{"hasImage": termQueryParameter{Value: q.HasImage}}})
	}
	if q.HasVideo.Valid {
		musts = append(musts, searchQuery{"term": termQuery{"hasVideo": termQueryParameter{Value: q.HasVideo}}})
	}
	if q.HasAudio.Valid {
		musts = append(musts, searchQuery{"term": termQuery{"hasAudio": termQueryParameter{Value: q.HasAudio}}})
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

	// ベクトル検索が可能かチェックし、ハイブリッド検索を試みる
	if q.Word.Valid && e.imageClient.Available() && e.imageConfig.VectorDimension > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), e.imageConfig.Timeout)
		defer cancel()

		queryVector, err := e.imageClient.EmbedText(ctx, q.Word.V)
		if err == nil && queryVector != nil {
			return e.doHybridSearch(musts, queryVector, sort, limit, offset)
		}
		e.l.Debug("falling back to text-only search", zap.Error(err))
	}

	return e.doTextSearch(musts, sort, limit, offset)
}

// doTextSearch 通常のテキスト検索を実行する
func (e *esEngine) doTextSearch(musts []searchQuery, sort string, limit, offset int) (Result, error) {
	b, err := json.Marshal(newSearchBody(musts))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal search query: %w", err)
	}

	sr, err := e.client.Search(
		e.client.Search.WithIndex(getIndexName(esMessageIndex)),
		e.client.Search.WithBody(bytes.NewBuffer(b)),
		e.client.Search.WithSort(sort),
		e.client.Search.WithSize(limit),
		e.client.Search.WithFrom(offset),
	)
	if err != nil {
		return nil, err
	}
	if sr.IsError() {
		return nil, fmt.Errorf("failed to get search result")
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

// doHybridSearch RRFによるハイブリッド検索（テキスト + ベクトル）を実行する
func (e *esEngine) doHybridSearch(musts []searchQuery, queryVector []float64, sort string, limit, offset int) (Result, error) {
	// RRF (Reciprocal Rank Fusion) を使ったハイブリッド検索
	// テキスト検索クエリ
	textQuery := newSearchBody(musts)

	// knnクエリ（nested field用）
	knnQuery := m{
		"field":          "imageVector.vector",
		"query_vector":   queryVector,
		"k":              limit + offset,
		"num_candidates": 100,
		"nested": m{
			"path": "imageVector",
		},
	}

	hybridBody := m{
		"query": textQuery.Query,
		"knn":   knnQuery,
		"rank": m{
			"rrf": m{},
		},
		"size": limit,
		"from": offset,
	}

	b, err := json.Marshal(hybridBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal hybrid search query: %w", err)
	}

	sr, err := e.client.Search(
		e.client.Search.WithIndex(getIndexName(esMessageIndex)),
		e.client.Search.WithBody(bytes.NewBuffer(b)),
	)
	if err != nil {
		// ハイブリッド検索に失敗した場合はテキスト検索にフォールバック
		e.l.Warn("hybrid search failed, falling back to text search", zap.Error(err))
		return e.doTextSearch(musts, sort, limit, offset)
	}
	if sr.IsError() {
		e.l.Warn("hybrid search returned error, falling back to text search")
		sr.Body.Close()
		return e.doTextSearch(musts, sort, limit, offset)
	}
	defer sr.Body.Close()

	var res esSearchResponse
	err = json.NewDecoder(sr.Body).Decode(&res)
	if err != nil {
		return nil, err
	}

	e.l.Debug("hybrid search result", zap.Reflect("hits", res.Hits))
	return e.parseResultFromResponse(res)
}

// generateImageURL ファイルの署名付きURLを生成する
func (e *esEngine) generateImageURL(fileID uuid.UUID) (string, error) {
	return e.fs.GenerateAccessURL(fileID.String(), model.FileTypeUserFile)
}

func (e *esEngine) Available() bool {
	// このクライアントにはライフサイクルが無いので、常にtrueを返す。
	return true
}

func (e *esEngine) Close() error {
	e.done <- struct{}{}
	return nil
}
