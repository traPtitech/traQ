package notification

import (
	"github.com/leandro-lugaresi/hub"
	"go.uber.org/zap"

	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/service/channel"
	"github.com/traPtitech/traQ/service/fcm"
	"github.com/traPtitech/traQ/service/file"
	"github.com/traPtitech/traQ/service/message"
	"github.com/traPtitech/traQ/service/search"
	"github.com/traPtitech/traQ/service/variable"
	"github.com/traPtitech/traQ/service/viewer"
	"github.com/traPtitech/traQ/service/ws"
)

// Service 通知サービス
type Service struct {
	repo   repository.Repository
	cm     channel.Manager
	mm     message.Manager
	fm     file.Manager
	hub    *hub.Hub
	logger *zap.Logger
	fcm    fcm.Client
	ws     *ws.Streamer
	vm     *viewer.Manager
	origin string
	search search.Engine
}

// NewService 通知サービスを作成して起動します
func NewService(repo repository.Repository, cm channel.Manager, mm message.Manager, fm file.Manager, hub *hub.Hub, logger *zap.Logger, fcm fcm.Client, ws *ws.Streamer, vm *viewer.Manager, origin variable.ServerOriginString, search search.Engine) *Service {
	service := &Service{
		repo:   repo,
		cm:     cm,
		mm:     mm,
		fm:     fm,
		hub:    hub,
		logger: logger.Named("notification"),
		fcm:    fcm,
		ws:     ws,
		vm:     vm,
		origin: string(origin),
		search: search,
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
