// +build wireinject

package cmd

import (
	"github.com/google/wire"
	"github.com/jinzhu/gorm"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router"
	"github.com/traPtitech/traQ/service"
	"github.com/traPtitech/traQ/service/bot"
	"github.com/traPtitech/traQ/service/channel"
	"github.com/traPtitech/traQ/service/counter"
	"github.com/traPtitech/traQ/service/file"
	"github.com/traPtitech/traQ/service/imaging"
	"github.com/traPtitech/traQ/service/notification"
	rbac2 "github.com/traPtitech/traQ/service/rbac"
	"github.com/traPtitech/traQ/service/viewer"
	"github.com/traPtitech/traQ/service/webrtcv3"
	"github.com/traPtitech/traQ/service/ws"
	"github.com/traPtitech/traQ/utils/storage"
	"go.uber.org/zap"
)

func newServer(hub *hub.Hub, db *gorm.DB, repo repository.Repository, fs storage.FileStorage, logger *zap.Logger, c *Config) (*Server, error) {
	wire.Build(
		bot.NewService,
		channel.InitChannelManager,
		file.InitFileManager,
		counter.NewOnlineCounter,
		counter.NewUnreadMessageCounter,
		counter.NewMessageCounter,
		counter.NewChannelCounter,
		imaging.NewProcessor,
		notification.NewService,
		rbac2.New,
		viewer.NewManager,
		webrtcv3.NewManager,
		ws.NewStreamer,
		router.Setup,
		newFCMClientIfAvailable,
		provideServerOriginString,
		provideFirebaseCredentialsFilePathString,
		provideImageProcessorConfig,
		provideRouterConfig,
		wire.Struct(new(service.Services), "*"),
		wire.Struct(new(Server), "*"),
		wire.Bind(new(repository.ChannelRepository), new(repository.Repository)),
		wire.Bind(new(repository.FileRepository), new(repository.Repository)),
	)
	return nil, nil
}
