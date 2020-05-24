package notification

import (
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/service/fcm"
	"github.com/traPtitech/traQ/service/sse"
	"github.com/traPtitech/traQ/service/variable"
	"github.com/traPtitech/traQ/service/viewer"
	"github.com/traPtitech/traQ/service/ws"
	"go.uber.org/zap"
)

// Service 通知サービス
type Service struct {
	repo   repository.Repository
	hub    *hub.Hub
	logger *zap.Logger
	fcm    fcm.Client
	sse    *sse.Streamer
	ws     *ws.Streamer
	vm     *viewer.Manager
	origin string
}

// NewService 通知サービスを作成して起動します
func NewService(repo repository.Repository, hub *hub.Hub, logger *zap.Logger, fcm fcm.Client, sse *sse.Streamer, ws *ws.Streamer, vm *viewer.Manager, origin variable.ServerOriginString) *Service {
	service := &Service{
		repo:   repo,
		hub:    hub,
		logger: logger.Named("notification"),
		fcm:    fcm,
		sse:    sse,
		ws:     ws,
		vm:     vm,
		origin: string(origin),
	}
	go func() {
		topics := make([]string, 0, len(handlerMap))
		for k := range handlerMap {
			topics = append(topics, k)
		}
		for msg := range hub.Subscribe(200, topics...).Receiver {
			h, ok := handlerMap[msg.Topic()]
			if ok {
				go h(service, msg)
			}
		}
	}()
	return service
}
