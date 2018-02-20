package router

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo"
	"github.com/labstack/gommon/log"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/notification"
)

// GetNotificationStatus GET /channels/:channelId/notifications のハンドラ
func GetNotificationStatus(c echo.Context) error {
	channelID := c.Param("channelID") //TODO チャンネルIDの検証

	users, err := model.GetSubscribingUser(uuid.FromStringOrNil(channelID))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf("failed to GetNotificationStatus: %v", err))
	}

	result := make([]string, len(users))
	for i, v := range users {
		result[i] = v.String()
	}

	return c.JSON(http.StatusOK, result)
}

// PutNotificationStatus PUT /channels/:channelId/notifications のハンドラ
func PutNotificationStatus(c echo.Context) error {
	channelID := c.Param("channelID") //TODO チャンネルIDの検証

	var req struct {
		On  []string `json:"on"`
		Off []string `json:"off"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	for _, v := range req.On {
		m := &model.UserSubscribeChannel{
			UserID:    v,
			ChannelID: channelID,
		}
		m.Create()
	}
	for _, v := range req.Off {
		m := &model.UserSubscribeChannel{
			UserID:    v,
			ChannelID: channelID,
		}
		m.Delete()
	}

	users, err := model.GetSubscribingUser(uuid.FromStringOrNil(channelID))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf("failed to GetNotificationStatus: %v", err))
	}

	result := make([]string, len(users))
	for i, v := range users {
		result[i] = v.String()
	}

	return c.JSON(http.StatusOK, result)
}

// PostDeviceToken POST /notification/device のハンドラ
func PostDeviceToken(c echo.Context) error {
	userID := c.Get("user").(*model.User).ID

	var req struct {
		Token string `json:"token"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	dev := &model.Device{
		UserID: userID,
		Token:  req.Token,
	}
	if err := dev.Register(); err != nil {
		log.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusCreated)
}

// GetNotificationStream GET /notification のハンドラ
func GetNotificationStream(c echo.Context) error {
	userID := uuid.FromStringOrNil(c.Get("user").(*model.User).ID)

	if _, ok := c.Response().Writer.(http.Flusher); !ok {
		return echo.NewHTTPError(http.StatusNotImplemented, "Server Sent Events is not supported.")
	}

	//Set headers for SSE
	c.Response().Header().Set(echo.HeaderContentType, "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")
	c.Response().WriteHeader(http.StatusOK)

	notification.Stream(userID, c.Response())
	return nil
}
