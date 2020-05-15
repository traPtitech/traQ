package v1

import (
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/service/viewer"
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
		return err
	}

	h.HeartBeats.Beat(userID, req.ChannelID, req.Status)

	return c.JSON(http.StatusOK, formatHeartbeat(req.ChannelID, viewer.ConvertToArray(h.VM.GetChannelViewers(req.ChannelID))))
}

// GetHeartbeat [deprecated] GET /heartbeat
func (h *Handlers) GetHeartbeat(c echo.Context) error {
	var req struct {
		ChannelID uuid.UUID `query:"channelId"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, formatHeartbeat(req.ChannelID, viewer.ConvertToArray(h.VM.GetChannelViewers(req.ChannelID))))
}
