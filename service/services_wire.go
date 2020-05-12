// +build wireinject

package service

import (
	"github.com/google/wire"
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

var ProviderSet = wire.NewSet(wire.FieldsOf(new(*Services),
	"BOT",
	"OnlineCounter",
	"FCM",
	"HeartBeats",
	"Imaging",
	"SSE",
	"ViewerManager",
	"WebRTCv3",
	"WS",
	"Notification",
))

func newServices(hub *hub.Hub, repo repository.Repository, fcm *fcm.Client, logger *zap.Logger, origin variable.ServerOriginString, imgConfig imaging.Config) *Services {
	wire.Build(
		bot.NewProcessor,
		counter.NewOnlineCounter,
		heartbeat.NewManager,
		imaging.NewProcessor,
		notification.NewService,
		sse.NewStreamer,
		viewer.NewManager,
		webrtcv3.NewManager,
		ws.NewStreamer,
		wire.Struct(new(Services), "*"),
	)
	return nil
}
