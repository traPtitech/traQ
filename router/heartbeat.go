package router

import (
	"github.com/labstack/echo"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/model"
	"net/http"
)

// PostHeartbeat POST /heartbeat
func PostHeartbeat(c echo.Context) error {
	req := struct {
		ChannelID uuid.UUID `json:"channelId"`
		Status    string    `json:"status"    validate:"required"`
	}{}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	userID := getRequestUserID(c)

	model.UpdateHeartbeatStatuses(userID, req.ChannelID, req.Status)

	status, _ := model.GetHeartbeatStatus(req.ChannelID)
	return c.JSON(http.StatusOK, status)
}

// GetHeartbeat GET /heartbeat
func GetHeartbeat(c echo.Context) error {
	req := struct {
		ChannelID string `query:"channelId" validate:"uuid"`
	}{}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	status, _ := model.GetHeartbeatStatus(uuid.FromStringOrNil(req.ChannelID))
	return c.JSON(http.StatusOK, status)
}
