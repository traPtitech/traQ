package router

import (
	"github.com/labstack/echo"
	"github.com/labstack/echo-contrib/session"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/notification"
	"net/http"
)

func GetNotificationStream(c echo.Context) error {

	//Authenticate
	s, err := session.Get("sessions", c)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized)
	}

	var userId uuid.UUID
	if id, ok := s.Values["userId"].(string); ok {
		userId = uuid.FromStringOrNil(id)
	}

	_, ok := c.Response().Writer.(http.Flusher)
	if !ok {
		return echo.NewHTTPError(http.StatusNotImplemented, "Server Sent Events is not supported.")
	}

	//Set headers for SSE

	c.Response().Header().Set(echo.HeaderContentType, "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")
	c.Response().WriteHeader(200)

	notification.SSEStreamer.Stream(userId, c.Response())

	return nil
}

