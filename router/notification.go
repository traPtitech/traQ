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

// GetNotification /channels/:ID/notificationsのpath paramがchannelIDかuserIDかを判別して正しいほうにルーティングするミドルウェア
func GetNotification(userHandler, channelHandler echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		userID := c.Get("user").(*model.User).ID
		ID := c.Param("ID")

		if ch, err := validateChannelID(ID, userID); ch != nil {
			c.Set("channel", ch)
			return channelHandler(c)
		} else if err != model.ErrNotFound {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to check ID")
		}

		if user, err := validateUserID(ID); user != nil {
			c.Set("targetUserID", user.ID)
			return userHandler(c)
		} else if err != model.ErrNotFound {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to check ID")
		}

		return echo.NewHTTPError(http.StatusBadRequest, "this ID does't exist")
	}
}

// GetNotificationStatus GET /channels/:channelID/notifications のハンドラ
func GetNotificationStatus(c echo.Context) error {
	ch := c.Get("channel").(*model.Channel)

	// プライベートチャンネルの通知は取得できない。
	if !ch.IsPublic {
		return echo.NewHTTPError(http.StatusForbidden)
	}

	users, err := model.GetSubscribingUser(uuid.FromStringOrNil(ch.ID))
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
	userID := c.Get("user").(*model.User).ID
	channelID := c.Param("ID")

	ch, err := validateChannelID(channelID, userID)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound, "this channel is not found")
		default:
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to find the specified channel")
		}
	}

	// プライベートチャンネルの通知は変更できない。
	if !ch.IsPublic {
		return echo.NewHTTPError(http.StatusForbidden)
	}

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

// GetNotificationChannels GET /users/{userID}/notification のハンドラ
func GetNotificationChannels(c echo.Context) error {
	userID := uuid.FromStringOrNil(c.Get("targetUserID").(string))

	channelIDs, err := model.GetSubscribedChannels(userID)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get subscribing channels")
	}

	res := make([]*ChannelForResponse, len(channelIDs))
	for i, v := range channelIDs {
		ch, err := model.GetChannelByID(userID.String(), v.String())
		if err != nil {
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get channels")
		}

		childIDs, err := ch.Children(userID.String())
		if err != nil {
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get children channel id list: %v", err)
		}

		var members []string
		if !ch.IsPublic {
			members, err = model.GetPrivateChannelMembers(ch.ID)
			if err != nil {
				c.Logger().Error(err)
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get private channel members")
			}
		}

		res[i] = formatChannel(ch, childIDs, members)
	}
	return c.JSON(http.StatusOK, res)
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
