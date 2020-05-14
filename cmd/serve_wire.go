// +build wireinject

package cmd

import (
	"github.com/google/wire"
	"github.com/jinzhu/gorm"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/rbac"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router"
	"github.com/traPtitech/traQ/service"
	"github.com/traPtitech/traQ/service/bot"
	"github.com/traPtitech/traQ/service/counter"
	"github.com/traPtitech/traQ/service/heartbeat"
	"github.com/traPtitech/traQ/service/imaging"
	"github.com/traPtitech/traQ/service/notification"
	"github.com/traPtitech/traQ/service/sse"
	"github.com/traPtitech/traQ/service/viewer"
	"github.com/traPtitech/traQ/service/webrtcv3"
	"github.com/traPtitech/traQ/service/ws"
	"go.uber.org/zap"
)

func newServer(hub *hub.Hub, db *gorm.DB, repo repository.Repository, logger *zap.Logger, r rbac.RBAC, c *Config) (*Server, error) {
	wire.Build(
		bot.NewProcessor,
		counter.NewOnlineCounter,
		counter.NewUnreadMessageCounter,
		counter.NewMessageCounter,
		counter.NewChannelCounter,
		heartbeat.NewManager,
		imaging.NewProcessor,
		notification.NewService,
		sse.NewStreamer,
		viewer.NewManager,
		webrtcv3.NewManager,
		ws.NewStreamer,
		router.Setup,
		newFCMClientIfAvailable,
		provideServerOriginString,
		provideFirebaseCredentialsFilePathString,
		provideImageProcessorConfig,
		providerRouterConfig,
		wire.Struct(new(service.Services), "*"),
		wire.Struct(new(Server), "*"),
	)
	return nil, nil
}
