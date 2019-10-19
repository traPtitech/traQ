package router

import (
	"github.com/gofrs/uuid"
	"github.com/labstack/echo"
	"net/http"
)

// PostHeartbeat POST /heartbeat
func (h *Handlers) PostHeartbeat(c echo.Context) error {
	userID := getRequestUserID(c)

	var req struct {
		ChannelID uuid.UUID `json:"channelId"`
		Status    string    `json:"status"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return badRequest(err)
	}

	h.Realtime.HeartBeats.Beat(userID, req.ChannelID, req.Status)

	return c.JSON(http.StatusOK, formatHeartbeat(req.ChannelID, h.Realtime.ViewerManager.GetChannelViewers(req.ChannelID)))
}

// GetHeartbeat GET /heartbeat
func (h *Handlers) GetHeartbeat(c echo.Context) error {
	var req struct {
		ChannelID uuid.UUID `query:"channelId"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return badRequest(err)
	}

	return c.JSON(http.StatusOK, formatHeartbeat(req.ChannelID, h.Realtime.ViewerManager.GetChannelViewers(req.ChannelID)))
}
