// +build wireinject

package router

import (
	"github.com/google/wire"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/oauth2"
	v1 "github.com/traPtitech/traQ/router/v1"
	v3 "github.com/traPtitech/traQ/router/v3"
	"github.com/traPtitech/traQ/service"
	"go.uber.org/zap"
)

func newRouter(hub *hub.Hub, repo repository.Repository, ss *service.Services, logger *zap.Logger, config *Config) *Router {
	wire.Build(
		service.ProviderSet,
		newEcho,
		provideOAuth2Config,
		provideV3Config,
		wire.Struct(new(v1.Handlers), "*"),
		wire.Struct(new(v3.Handlers), "*"),
		wire.Struct(new(oauth2.Handler), "*"),
		wire.Struct(new(Router), "*"),
	)
	return nil
}
