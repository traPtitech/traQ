package notification

import (
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/notification/fcm"
	"github.com/traPtitech/traQ/realtime"
	"github.com/traPtitech/traQ/realtime/sse"
	"github.com/traPtitech/traQ/realtime/ws"
	"github.com/traPtitech/traQ/repository"
	"go.uber.org/zap"
)

// Service 通知サービス
type Service struct {
	repo     repository.Repository
	hub      *hub.Hub
	logger   *zap.Logger
	fcm      *fcm.Client
	sse      *sse.Streamer
	ws       *ws.Streamer
	realtime *realtime.Service
	origin   string
}

// StartService 通知サービスを作成して起動します
func StartService(repo repository.Repository, hub *hub.Hub, logger *zap.Logger, fcm *fcm.Client, sse *sse.Streamer, ws *ws.Streamer, realtime *realtime.Service, origin string) *Service {
	service := &Service{
		repo:     repo,
		hub:      hub,
		logger:   logger,
		fcm:      fcm,
		sse:      sse,
		ws:       ws,
		realtime: realtime,
		origin:   origin,
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
