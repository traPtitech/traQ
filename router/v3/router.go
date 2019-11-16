package v3

import (
	"github.com/labstack/echo/v4"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/rbac"
	"github.com/traPtitech/traQ/rbac/permission"
	"github.com/traPtitech/traQ/realtime"
	"github.com/traPtitech/traQ/realtime/ws"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/middlewares"
	"go.uber.org/zap"
)

type Handlers struct {
	RBAC     rbac.RBAC
	Repo     repository.Repository
	WS       *ws.Streamer
	Hub      *hub.Hub
	Logger   *zap.Logger
	Realtime *realtime.Service

	Version  string
	Revision string
}

// Setup APIルーティングを行います
func (h *Handlers) Setup(e *echo.Group) {
	// middleware preparation
	requires := middlewares.AccessControlMiddlewareGenerator(h.RBAC)

	api := e.Group("/v3", middlewares.UserAuthenticate(h.Repo))
	{
		api.GET("/ws", echo.WrapHandler(h.WS), requires(permission.ConnectNotificationStream))
	}

	apiNoAuth := e.Group("/v3")
	{
		apiNoAuth.GET("/version", h.GetVersion)
	}
}
