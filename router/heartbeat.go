package router

import (
	"github.com/gofrs/uuid"
	"github.com/labstack/echo"
	"net/http"
)

// PostHeartbeat POST /heartbeat
func (h *Handlers) PostHeartbeat(c echo.Context) error {
	userID := getRequestUserID(c)

	req := struct {
		ChannelID uuid.UUID `json:"channelId"`
		Status    string    `json:"status"`
	}{}
	if err := bindAndValidate(c, &req); err != nil {
		return badRequest(err)
	}

	h.Repo.UpdateHeartbeatStatus(userID, req.ChannelID, req.Status)

	status, _ := h.Repo.GetHeartbeatStatus(req.ChannelID)
	return c.JSON(http.StatusOK, status)
}

// GetHeartbeat GET /heartbeat
func (h *Handlers) GetHeartbeat(c echo.Context) error {
	req := struct {
		ChannelID uuid.UUID `query:"channelId"`
	}{}
	if err := bindAndValidate(c, &req); err != nil {
		return badRequest(err)
	}

	status, _ := h.Repo.GetHeartbeatStatus(req.ChannelID)
	return c.JSON(http.StatusOK, status)
}
