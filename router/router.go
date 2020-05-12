package router

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/leandro-lugaresi/hub"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/traPtitech/traQ/rbac"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/auth"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension"
	"github.com/traPtitech/traQ/router/middlewares"
	"github.com/traPtitech/traQ/router/oauth2"
	"github.com/traPtitech/traQ/router/v1"
	"github.com/traPtitech/traQ/router/v3"
	"github.com/traPtitech/traQ/service"
	"go.uber.org/zap"
	"net/http"
)

type Router struct {
	e      *echo.Echo
	v1     *v1.Handlers
	v3     *v3.Handlers
	oauth2 *oauth2.Handler
}

func Setup(hub *hub.Hub, repo repository.Repository, ss *service.Services, rbac rbac.RBAC, logger *zap.Logger, config *Config) *echo.Echo {
	r := newRouter(hub, repo, ss, rbac, logger.Named("router"), config)

	api := r.e.Group("/api")
	api.GET("/metrics", echo.WrapHandler(promhttp.Handler()))
	api.GET("/ping", func(c echo.Context) error { return c.String(http.StatusOK, http.StatusText(http.StatusOK)) })
	r.v1.Setup(api)
	r.v3.Setup(api)
	r.oauth2.Setup(api.Group("/oauth2"))
	r.oauth2.Setup(api.Group("/1.0/oauth2"))
	r.oauth2.Setup(api.Group("/v3/oauth2"))

	// 外部authハンドラ
	extAuth := api.Group("/auth")
	if config.ExternalAuth.GitHub.Valid() {
		p := auth.NewGithubProvider(repo, logger.Named("ext_auth"), config.ExternalAuth.GitHub)
		extAuth.GET("/github", p.LoginHandler)
		extAuth.GET("/github/callback", p.CallbackHandler)
	}
	if config.ExternalAuth.Google.Valid() {
		p := auth.NewGoogleProvider(repo, logger.Named("ext_auth"), config.ExternalAuth.Google)
		extAuth.GET("/google", p.LoginHandler)
		extAuth.GET("/google/callback", p.CallbackHandler)
	}
	if config.ExternalAuth.TraQ.Valid() {
		p := auth.NewTraQProvider(repo, logger.Named("ext_auth"), config.ExternalAuth.TraQ)
		extAuth.GET("/traq", p.LoginHandler)
		extAuth.GET("/traq/callback", p.CallbackHandler)
	}
	if config.ExternalAuth.OIDC.Valid() {
		p, err := auth.NewOIDCProvider(repo, logger.Named("ext_auth"), config.ExternalAuth.OIDC)
		if err != nil {
			panic(err)
		}
		extAuth.GET("/oidc", p.LoginHandler)
		extAuth.GET("/oidc/callback", p.CallbackHandler)
	}

	return r.e
}

func newEcho(logger *zap.Logger, config *Config, repo repository.Repository) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.HTTPErrorHandler = extension.ErrorHandler(logger)

	// ミドルウェア設定
	e.Use(middlewares.ServerVersion(config.Version))
	e.Use(middlewares.RequestID())
	if config.AccessLogging {
		e.Use(middlewares.AccessLogging(logger.Named("access_log"), config.Development))
	}
	e.Use(middlewares.Recovery(logger))
	if config.Gzipped {
		e.Use(middlewares.Gzip())
	}
	e.Use(extension.Wrap(repo))
	e.Use(middlewares.RequestCounter())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		ExposeHeaders: []string{consts.HeaderVersion, consts.HeaderCacheFile, consts.HeaderFileMetaType, consts.HeaderMore, echo.HeaderXRequestID},
		AllowHeaders:  []string{echo.HeaderContentType, echo.HeaderAuthorization, consts.HeaderSignature},
		MaxAge:        3600,
	}))

	return e
}
