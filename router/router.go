package router

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension"
	"github.com/traPtitech/traQ/router/middlewares"
	"github.com/traPtitech/traQ/router/v1"
	"github.com/traPtitech/traQ/utils/validator"
)

// Setup APIサーバーハンドラを構築します
func Setup(config *Config) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Validator = validator.New()
	e.Binder = &extension.Binder{}
	e.HTTPErrorHandler = extension.ErrorHandler(config.RootLogger.Named("api_handler"))

	// ミドルウェア設定
	e.Use(middlewares.ServerVersion(config.Version))
	if config.AccessLogging {
		e.Use(middlewares.AccessLogging(config.RootLogger.Named("access_log")))
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

	// v1 APIハンドラ
	v1.LoadWebhookTemplate("static/webhook/*.tmpl")
	v1 := v1.Handlers{
		RBAC:             config.RBAC,
		Repo:             config.Repository,
		SSE:              config.SSE,
		WS:               config.WS,
		Hub:              config.Hub,
		Logger:           config.RootLogger.Named("api_handler"),
		Realtime:         config.Realtime,
		ImageMagickPath:  config.ImageMagickPath,
		AccessTokenExp:   config.AccessTokenExp,
		IsRefreshEnabled: config.IsRefreshEnabled,
		SkyWaySecretKey:  config.SkyWaySecretKey,
	}
	v1.Setup(api)

	return e
}
