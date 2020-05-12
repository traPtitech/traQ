package service

import (
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/service/bot"
	"github.com/traPtitech/traQ/service/counter"
	"github.com/traPtitech/traQ/service/fcm"
	"github.com/traPtitech/traQ/service/heartbeat"
	"github.com/traPtitech/traQ/service/imaging"
	"github.com/traPtitech/traQ/service/notification"
	"github.com/traPtitech/traQ/service/sse"
	"github.com/traPtitech/traQ/service/variable"
	"github.com/traPtitech/traQ/service/viewer"
	"github.com/traPtitech/traQ/service/webrtcv3"
	"github.com/traPtitech/traQ/service/ws"
	"go.uber.org/zap"
)

type Services struct {
	BOT           *bot.Processor
	OnlineCounter *counter.OnlineCounter
	FCM           *fcm.Client
	HeartBeats    *heartbeat.Manager
	Imaging       imaging.Processor
	SSE           *sse.Streamer
	ViewerManager *viewer.Manager
	WebRTCv3      *webrtcv3.Manager
	WS            *ws.Streamer
	Notification  *notification.Service
}

func NewServices(hub *hub.Hub, repo repository.Repository, fcm *fcm.Client, logger *zap.Logger, origin variable.ServerOriginString, imgConfig imaging.Config) *Services {
	return newServices(hub, repo, fcm, logger, origin, imgConfig)
}

func (ss *Services) Dispose() {
	ss.SSE.Dispose()
	_ = ss.WS.Close()
	if ss.FCM != nil {
		ss.FCM.Close()
	}
}
