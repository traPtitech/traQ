package router

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/traPtitech/traQ/router/auth"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension"
	"github.com/traPtitech/traQ/router/middlewares"
	"github.com/traPtitech/traQ/router/oauth2"
	"github.com/traPtitech/traQ/router/utils"
	"github.com/traPtitech/traQ/router/v1"
	v3 "github.com/traPtitech/traQ/router/v3"
	"net/http"
)

// Setup APIサーバーハンドラを構築します
func Setup(config *Config) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.HTTPErrorHandler = extension.ErrorHandler(config.RootLogger.Named("api_handler"))

	// ミドルウェア設定
	e.Use(middlewares.ServerVersion(config.Version))
	if config.AccessLogging {
		e.Use(middlewares.AccessLogging(config.RootLogger.Named("access_log"), config.Development))
	}
	if config.Gzipped {
		e.Use(middlewares.Gzip())
	}
	e.Use(extension.Wrap())
	e.Use(middlewares.RequestCounter())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		ExposeHeaders: []string{consts.HeaderVersion, consts.HeaderCacheFile, consts.HeaderFileMetaType, consts.HeaderMore},
		AllowHeaders:  []string{echo.HeaderContentType, echo.HeaderAuthorization, consts.HeaderSignature},
		MaxAge:        3600,
	}))

	api := e.Group("/api")
	api.GET("/metrics", echo.WrapHandler(promhttp.Handler()))
	api.GET("/ping", func(c echo.Context) error { return c.String(http.StatusOK, http.StatusText(http.StatusOK)) })

	// v1 APIハンドラ
	v1 := v1.Handlers{
		RBAC:            config.RBAC,
		Repo:            config.Repository,
		SSE:             config.SSE,
		WS:              config.WS,
		Hub:             config.Hub,
		Logger:          config.RootLogger.Named("api_handler"),
		Realtime:        config.Realtime,
		SkyWaySecretKey: config.SkyWaySecretKey,
	}
	v1.Setup(api)

	// v3 APIハンドラ
	v3 := v3.Handlers{
		RBAC:            config.RBAC,
		Repo:            config.Repository,
		WS:              config.WS,
		Hub:             config.Hub,
		Logger:          config.RootLogger.Named("api_handler"),
		Realtime:        config.Realtime,
		Version:         config.Version,
		Revision:        config.Revision,
		SkyWaySecretKey: config.SkyWaySecretKey,
	}
	v3.Setup(api)

	// oauth2ハンドラ
	oa2 := &oauth2.Config{
		RBAC:             config.RBAC,
		Repo:             config.Repository,
		Logger:           config.RootLogger.Named("oauth2_api_handler"),
		AccessTokenExp:   config.AccessTokenExp,
		IsRefreshEnabled: config.IsRefreshEnabled,
	}
	oa2.Setup(api.Group("/oauth2"))
	oa2.Setup(api.Group("/1.0/oauth2"))
	oa2.Setup(api.Group("/v3/oauth2"))

	// 外部authハンドラ
	extAuth := api.Group("/auth")
	if config.ExternalAuth.GitHub.Valid() {
		p := auth.NewGithubProvider(config.Repository, config.RootLogger.Named("ext_auth"), config.ExternalAuth.GitHub)
		extAuth.GET("/github", p.LoginHandler)
		extAuth.GET("/github/callback", p.CallbackHandler)
	}
	if config.ExternalAuth.Google.Valid() {
		p := auth.NewGoogleProvider(config.Repository, config.RootLogger.Named("ext_auth"), config.ExternalAuth.Google)
		extAuth.GET("/google", p.LoginHandler)
		extAuth.GET("/google/callback", p.CallbackHandler)
	}
	if config.ExternalAuth.TraQ.Valid() {
		p := auth.NewTraQProvider(config.Repository, config.RootLogger.Named("ext_auth"), config.ExternalAuth.TraQ)
		extAuth.GET("/traq", p.LoginHandler)
		extAuth.GET("/traq/callback", p.CallbackHandler)
	}

	utils.ImageMagickPath = config.ImageMagickPath
	return e
}
