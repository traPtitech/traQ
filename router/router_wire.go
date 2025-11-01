//go:build wireinject

package router

import (
	"github.com/google/wire"
	"github.com/leandro-lugaresi/hub"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/oauth2"
	"github.com/traPtitech/traQ/router/session"
	"github.com/traPtitech/traQ/router/utils"
	v1 "github.com/traPtitech/traQ/router/v1"
	v3 "github.com/traPtitech/traQ/router/v3"
	"github.com/traPtitech/traQ/service"
	"github.com/traPtitech/traQ/utils/message"
)

func newRouter(hub *hub.Hub, db *gorm.DB, repo repository.Repository, ss *service.Services, logger *zap.Logger, config *Config) *Router {
	wire.Build(
		service.ProviderSet,
		newEcho,
		utils.NewReplaceMapper,
		message.NewReplacer,
		v1.NewEmojiCache,
		provideOAuth2Config,
		provideV3Config,
		session.NewGormStore,
		wire.Struct(new(v1.Handlers), "*"),
		wire.Struct(new(v3.Handlers), "*"),
		wire.Struct(new(oauth2.Handler), "*"),
		wire.Struct(new(Router), "*"),
	)
	return nil
}
