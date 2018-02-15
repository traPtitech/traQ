package router

import (
	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/model"
	"net/http"
)

// PostHeartbeat POST /heartbeat のハンドラ
func PostHeartbeat(c echo.Context) error {
	userID := c.Get("user").(*model.User).ID

	requestBody := struct {
		ChannelID string `json:"channelId"`
		Status    string `json:"status"`
	}{}

	if err := c.Bind(&requestBody); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Failed to bind request body.")
	}
	model.UpdateHeartbeatStatuses(userID, requestBody.ChannelID, requestBody.Status)

	status, _ := model.GetHeartbeatStatus(requestBody.ChannelID)
	return c.JSON(http.StatusOK, status)
}

// GetHeartbeat GET /heartbeat のハンドラ
func GetHeartbeat(c echo.Context) error {
	requestBody := struct {
		ChannelID string `query:"channelId"`
	}{}
	if err := c.Bind(&requestBody); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Failed to bind request query")
	}

	status, _ := model.GetHeartbeatStatus(requestBody.ChannelID)
	return c.JSON(http.StatusOK, status)
}
