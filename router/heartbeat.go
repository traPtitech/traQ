package router

import (
	"github.com/labstack/echo"
	"github.com/satori/go.uuid"
	"net/http"
)

// PostHeartbeat POST /heartbeat
func (h *Handlers) PostHeartbeat(c echo.Context) error {
	userID := getRequestUserID(c)

	req := struct {
		ChannelID uuid.UUID `json:"channelId"`
		Status    string    `json:"status"    validate:"required"`
	}{}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
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
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	status, _ := h.Repo.GetHeartbeatStatus(req.ChannelID)
	return c.JSON(http.StatusOK, status)
}
