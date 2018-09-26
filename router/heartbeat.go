package router

import (
	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/model"
	"net/http"
)

// PostHeartbeat POST /heartbeat
func PostHeartbeat(c echo.Context) error {
	user := getRequestUser(c)

	req := struct {
		ChannelID string `json:"channelId" validate:"uuid"`
		Status    string `json:"status"    validate:"required"`
	}{}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	model.UpdateHeartbeatStatuses(user.ID, req.ChannelID, req.Status)

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

	status, _ := model.GetHeartbeatStatus(req.ChannelID)
	return c.JSON(http.StatusOK, status)
}
