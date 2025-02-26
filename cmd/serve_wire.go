//go:build wireinject
// +build wireinject

package cmd

import (
	"github.com/google/wire"
	"github.com/leandro-lugaresi/hub"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router"
	"github.com/traPtitech/traQ/service"
	"github.com/traPtitech/traQ/service/bot"
	botWS "github.com/traPtitech/traQ/service/bot/ws"
	"github.com/traPtitech/traQ/service/channel"
	"github.com/traPtitech/traQ/service/counter"
	"github.com/traPtitech/traQ/service/exevent"
	"github.com/traPtitech/traQ/service/file"
	"github.com/traPtitech/traQ/service/imaging"
	"github.com/traPtitech/traQ/service/message"
	"github.com/traPtitech/traQ/service/notification"
	"github.com/traPtitech/traQ/service/ogp"
	rbac2 "github.com/traPtitech/traQ/service/rbac"
	"github.com/traPtitech/traQ/service/viewer"
	"github.com/traPtitech/traQ/service/webrtcv3"
	"github.com/traPtitech/traQ/service/ws"
	"github.com/traPtitech/traQ/utils/storage"
)

func newServer(hub *hub.Hub, db *gorm.DB, repo repository.Repository, fs storage.FileStorage, logger *zap.Logger, c *Config) (*Server, error) {
	wire.Build(
		bot.NewService,
		channel.InitChannelManager,
		file.InitFileManager,
		message.NewMessageManager,
		counter.NewOnlineCounter,
		counter.NewUnreadMessageCounter,
		counter.NewMessageCounter,
		counter.NewUserCounter,
		counter.NewChannelCounter,
		exevent.NewStampThrottler,
		imaging.NewProcessor,
		notification.NewService,
		ogp.NewServiceImpl,
		rbac2.New,
		viewer.NewManager,
		webrtcv3.NewManager,
		ws.NewStreamer,
		botWS.NewStreamer,
		router.Setup,
		newFCMClientIfAvailable,
		initSearchServiceIfAvailable,
		provideServerOriginString,
		provideFirebaseCredentialsFilePathString,
		provideImageProcessorConfig,
		provideOIDCService,
		provideRouterConfig,
		provideESEngineConfig,
		wire.Struct(new(service.Services), "*"),
		wire.Struct(new(Server), "*"),
		wire.Bind(new(repository.ChannelRepository), new(repository.Repository)),
		wire.Bind(new(repository.FileRepository), new(repository.Repository)),
	)
	return nil, nil
}
