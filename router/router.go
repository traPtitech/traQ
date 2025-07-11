package router

import (
	"net/http"

	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/leandro-lugaresi/hub"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/auth"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension"
	"github.com/traPtitech/traQ/router/middlewares"
	"github.com/traPtitech/traQ/router/oauth2"
	"github.com/traPtitech/traQ/router/session"
	v1 "github.com/traPtitech/traQ/router/v1"
	v3 "github.com/traPtitech/traQ/router/v3"
	"github.com/traPtitech/traQ/service"
	"github.com/traPtitech/traQ/service/channel"
)

type Router struct {
	e         *echo.Echo
	sessStore session.Store
	v1        *v1.Handlers
	v3        *v3.Handlers
	oauth2    *oauth2.Handler
}

func Setup(hub *hub.Hub, db *gorm.DB, repo repository.Repository, ss *service.Services, logger *zap.Logger, config *Config) *echo.Echo {
	r := newRouter(hub, db, repo, ss, logger.Named("router"), config)

	wellKnown := r.e.Group("/.well-known")
	{
		wellKnown.GET("/reset-password", func(c echo.Context) error {
			return c.Redirect(http.StatusFound, "/settings/session")
		})
		wellKnown.GET("/openid-configuration", r.oauth2.OIDCDiscovery)
		wellKnown.GET("/security.txt", func(c echo.Context) error {
			// Contactなどの情報が古くなっていないかを定期的に確認し、Expiresを*手動で*更新すること
			// See: https://www.rfc-editor.org/rfc/rfc9116.html#section-5.3
			return c.String(http.StatusOK, `Contact: mailto:info@trap.jp
Contact: https://trap.jp/request
Expires: 2026-03-31T23:59:59+09:00
Preferred-Languages: ja,en`)
		})
	}

	api := r.e.Group("/api")
	api.GET("/metrics", echoprometheus.NewHandler())
	api.GET("/ping", func(c echo.Context) error { return c.String(http.StatusOK, http.StatusText(http.StatusOK)) })
	r.v1.Setup(api)
	r.v3.Setup(api)
	r.oauth2.Setup(api.Group("/oauth2"))
	r.oauth2.Setup(api.Group("/v3/oauth2"))

	// 外部authハンドラ
	extAuth := api.Group("/auth")
	if config.ExternalAuth.GitHub.Valid() {
		p := auth.NewGithubProvider(repo, ss.FileManager, logger.Named("ext_auth"), r.sessStore, config.ExternalAuth.GitHub)
		extAuth.GET("/github", p.LoginHandler)
		extAuth.GET("/github/callback", p.CallbackHandler)
	}
	if config.ExternalAuth.Google.Valid() {
		p := auth.NewGoogleProvider(repo, ss.FileManager, logger.Named("ext_auth"), r.sessStore, config.ExternalAuth.Google)
		extAuth.GET("/google", p.LoginHandler)
		extAuth.GET("/google/callback", p.CallbackHandler)
	}
	if config.ExternalAuth.TraQ.Valid() {
		p := auth.NewTraQProvider(repo, ss.FileManager, logger.Named("ext_auth"), r.sessStore, config.ExternalAuth.TraQ)
		extAuth.GET("/traq", p.LoginHandler)
		extAuth.GET("/traq/callback", p.CallbackHandler)
	}
	if config.ExternalAuth.OIDC.Valid() {
		p, err := auth.NewOIDCProvider(repo, ss.FileManager, logger.Named("ext_auth"), r.sessStore, config.ExternalAuth.OIDC)
		if err != nil {
			panic(err)
		}
		extAuth.GET("/oidc", p.LoginHandler)
		extAuth.GET("/oidc/callback", p.CallbackHandler)
	}
	if config.ExternalAuth.Slack.Valid() {
		p := auth.NewSlackProvider(repo, ss.FileManager, logger.Named("ext_auth"), r.sessStore, config.ExternalAuth.Slack)
		extAuth.GET("/slack", p.LoginHandler)
		extAuth.GET("/slack/callback", p.CallbackHandler)
	}

	return r.e
}

func newEcho(logger *zap.Logger, config *Config, repo repository.Repository, cm channel.Manager) *echo.Echo {
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
	e.Use(extension.Wrap(repo, cm))
	e.Use(middlewares.RequestCounter())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		ExposeHeaders: []string{consts.HeaderVersion, consts.HeaderCacheFile, consts.HeaderFileMetaType, consts.HeaderMore, echo.HeaderXRequestID},
		AllowHeaders:  []string{echo.HeaderContentType, echo.HeaderAuthorization, consts.HeaderSignature, consts.HeaderChannelID},
		MaxAge:        3600,
	}))
	e.Use(echoprometheus.NewMiddleware("echo"))

	return e
}
