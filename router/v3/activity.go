package v3

import (
	"github.com/labstack/echo/v4"
	"net/http"
)

// GetOnlineUsers GET /activity/onlines
func (h *Handlers) GetOnlineUsers(c echo.Context) error {
	return c.JSON(http.StatusOK, h.Realtime.OnlineCounter.GetOnlineUserIDs())
}
