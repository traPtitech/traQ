package notification

import (
	"github.com/leandro-lugaresi/hub"
	"go.uber.org/zap"

	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/service/channel"
	"github.com/traPtitech/traQ/service/fcm"
	"github.com/traPtitech/traQ/service/file"
	"github.com/traPtitech/traQ/service/message"
	"github.com/traPtitech/traQ/service/variable"
	"github.com/traPtitech/traQ/service/viewer"
	"github.com/traPtitech/traQ/service/ws"
)

type messageWriter interface {
	WriteMessage(string, interface{}, ws.TargetFunc)
}

// Service 通知サービス
type Service struct {
	repo   repository.Repository
	cm     channel.Manager
	mm     message.Manager
	fm     file.Manager
	hub    *hub.Hub
	logger *zap.Logger
	fcm    fcm.Client
	ws     messageWriter
	vm     *viewer.Manager
	origin string
}

// NewService 通知サービスを作成して起動します
func NewService(repo repository.Repository, cm channel.Manager, mm message.Manager, fm file.Manager, hub *hub.Hub, logger *zap.Logger, fcm fcm.Client, ws *ws.Streamer, vm *viewer.Manager, origin variable.ServerOriginString) *Service {
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
	}

	service.startPresenceNotifications()
	service.startConcurrentNotifications()
	return service
}
