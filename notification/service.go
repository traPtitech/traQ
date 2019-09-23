package notification

import (
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/fcm"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/sse"
	"go.uber.org/zap"
)

// Service 通知サービス
type Service struct {
	repo   repository.Repository
	hub    *hub.Hub
	logger *zap.Logger
	fcm    *fcm.Client
	sse    *sse.Streamer
	origin string
}

// StartService 通知サービスを作成して起動します
func StartService(repo repository.Repository, hub *hub.Hub, logger *zap.Logger, fcm *fcm.Client, sse *sse.Streamer, origin string) *Service {
	service := &Service{
		repo:   repo,
		hub:    hub,
		logger: logger,
		fcm:    fcm,
		sse:    sse,
		origin: origin,
	}
	go func() {
		for msg := range hub.Subscribe(200, "*").Receiver {
			h, ok := handlerMap[msg.Topic()]
			if ok {
				go h(service, msg)
			}
		}
	}()
	return service
}
