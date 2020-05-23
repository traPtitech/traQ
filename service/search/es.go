package search

import (
	"context"
	"fmt"
	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"
	"github.com/olivere/elastic/v7"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"go.uber.org/zap"
)

const (
	esRequiredVersion = "7.7.0"
	esIndexPrefix     = "traq_"
	esMessageIndex    = "message"
)

type m map[string]interface{}

type ESURLString string

type esEngine struct {
	client *elastic.Client
	hub    *hub.Hub
	l      *zap.Logger
}

func NewESEngine(hub *hub.Hub, logger *zap.Logger, url ESURLString) (Engine, error) {
	// es接続
	client, err := elastic.NewClient(elastic.SetURL(string(url)))
	if err != nil {
		return nil, fmt.Errorf("failed to init search engine: %w", err)
	}

	// esバージョン確認
	version, err := client.ElasticsearchVersion(string(url))
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
					"type": "date",
				},
				"updatedAt": m{
					"type": "date",
				},
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
		l:      logger.Named("search"),
	}

	go func() {
		for ev := range hub.Subscribe(10, event.MessageCreated, event.MessageUpdated, event.MessageDeleted).Receiver {
			engine.onEvent(ev)
		}
	}()
	return engine, nil
}

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

	}
}

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

func (e *esEngine) deleteMessageFromIndex(id uuid.UUID) error {
	_, err := e.client.Delete().
		Index(getIndexName(esMessageIndex)).
		Id(id.String()).
		Do(context.Background())
	return err
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
