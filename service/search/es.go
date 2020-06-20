package search

import (
	"context"
	"fmt"
	"github.com/gofrs/uuid"
	json "github.com/json-iterator/go"
	"github.com/leandro-lugaresi/hub"
	"github.com/olivere/elastic/v7"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"go.uber.org/zap"
	"time"
)

const (
	esRequiredVersion = "7.7.0"
	esIndexPrefix     = "traq_"
	esMessageIndex    = "message"
)

type m map[string]interface{}

// ESEngineConfig Elasticsearch検索エンジン設定
type ESEngineConfig struct {
	// URL ESのURL
	URL string
}

type esEngine struct {
	client *elastic.Client
	hub    *hub.Hub
	repo   repository.Repository
	l      *zap.Logger
}

type esMessageDoc struct {
	ID        uuid.UUID `json:"-"`
	UserID    uuid.UUID `json:"userId"`
	ChannelID uuid.UUID `json:"channelId"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type esResult struct {
	docs []*esMessageDoc
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
		r2, err := client.PutMapping().Index(getIndexName(esMessageIndex)).BodyJson(m{
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
					"format": "yyyy-MM-ddTHH:mm:ssZ",
				},
				"updatedAt": m{
					"type":   "date",
					"format": "yyyy-MM-ddTHH:mm:ssZ",
				},
				// TODO To(複数)をメッセージ投稿時に、Cite(複数)をメッセージ投稿・更新時にindexに追加
				// TODO textをパースしてTo, Cite, Is*, Has*の判定
			},
		}).Do(context.Background())
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

	go func() {
		for ev := range hub.Subscribe(10, event.MessageCreated, event.MessageUpdated, event.MessageDeleted).Receiver {
			engine.onEvent(ev)
		}
	}()
	return engine, nil
}

// onEvent 内部イベントを処理する
func (e *esEngine) onEvent(ev hub.Message) {
	switch ev.Topic() {
	case event.MessageCreated:
		err := e.addMessageToIndex(ev.Fields["message"].(*model.Message))
		if err != nil {
			e.l.Error(err.Error(), zap.Error(err))
		}

	case event.MessageUpdated:
		err := e.updateMessageOnIndex(ev.Fields["message"].(*model.Message))
		if err != nil {
			e.l.Error(err.Error(), zap.Error(err))
		}

	case event.MessageDeleted:
		err := e.deleteMessageFromIndex(ev.Fields["message_id"].(uuid.UUID))
		if err != nil {
			e.l.Error(err.Error(), zap.Error(err))
		}

		// スタンプの追加・削除は優先度低め
	}
}

// addMessageToIndex 新規メッセージをesに入れる
func (e *esEngine) addMessageToIndex(m *model.Message) error {
	_, err := e.client.Index().
		Index(getIndexName(esMessageIndex)).
		Id(m.ID.String()).
		BodyJson(map[string]interface{}{
			"userId":    m.UserID,
			"channelId": m.ChannelID,
			"text":      m.Text,
			"createdAt": m.CreatedAt,
			"updatedAt": m.UpdatedAt,
		}).Do(context.Background())
	if err != nil {
		return err
	}
	return nil
}

// updateMessageOnIndex 既存メッセージの編集をesに反映させる
func (e *esEngine) updateMessageOnIndex(m *model.Message) error {
	_, err := e.client.Update().
		Index(getIndexName(esMessageIndex)).
		Id(m.ID.String()).
		Doc(map[string]interface{}{
			"text":      m.Text,
			"updatedAt": m.UpdatedAt,
		}).Do(context.Background())
	return err
}

// deleteMessageFromIndex メッセージの削除をesに反映させる
func (e *esEngine) deleteMessageFromIndex(id uuid.UUID) error {
	_, err := e.client.Delete().
		Index(getIndexName(esMessageIndex)).
		Id(id.String()).
		Do(context.Background())
	return err
}

func (e *esEngine) Do(q *Query) (Result, error) {
	// TODO 実装
	e.l.Debug("do search", zap.Reflect("q", q))

	fmt.Println(q.Word)
	fmt.Println(q.After)

	// TODO "should" "must not"をどういれるか
	musts := []elastic.Query{elastic.NewMatchPhraseQuery("text", q.Word)} // TODO MatchPhraseQuery(語順が重要な場合)との出し分け

	switch {
	case q.After.Valid && q.Before.Valid:
		musts = append(musts, elastic.NewRangeQuery("date").Gte(q.After).Lte(q.Before).Format("yyyy-MM-ddTHH:mm:ssZ"))
	case q.After.Valid && !q.Before.Valid:
		musts = append(musts, elastic.NewRangeQuery("date").Gte(q.After).Format("yyyy-MM-ddTHH:mm:ssZ"))
	case !q.After.Valid && q.Before.Valid:
		musts = append(musts, elastic.NewRangeQuery("date").Lte(q.Before).Format("yyyy-MM-ddTHH:mm:ssZ"))
	}

	switch {
	case q.To.Valid:
		musts = append(musts, elastic.NewMatchQuery("text", q.To))
	case q.From.Valid:
		musts = append(musts, elastic.NewMatchQuery("userId", q.From))
	}

	if q.Cite.Valid {
		musts = append(musts, elastic.NewMatchQuery("text", q.Cite))
	}

	// TODO
	//IsEdited
	//IsCited
	//IsPinned
	//HasURL
	//HasEmbedded
	//HasImage
	//HasMovie
	//HasAudio

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

func getIndexName(index string) string {
	return esIndexPrefix + index
}
