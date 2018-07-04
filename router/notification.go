package router

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/model"
)

// GetNotification /channels/:ID/notificationsのpath paramがchannelIDかuserIDかを判別して正しいほうにルーティングするミドルウェア
func GetNotification(userHandler, channelHandler echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		userID := c.Get("user").(*model.User).GetUID()
		ID := uuid.FromStringOrNil(c.Param("ID"))

		if ch, err := validateChannelID(ID, userID); ch != nil {
			c.Set("channel", ch)
			return channelHandler(c)
		} else if err != model.ErrNotFound {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to check ID")
		}

		if user, err := model.GetUser(userID); user != nil {
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

	users, err := model.GetSubscribingUser(ch.GetCID())
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
	userID := c.Get("user").(*model.User).GetUID()
	channelID := uuid.FromStringOrNil(c.Param("ID"))

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
		On  []string `json:"on"  validate:"dive,uuid"`
		Off []string `json:"off" validate:"dive,uuid"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	chID := ch.GetCID()
	for _, v := range req.On {
		model.SubscribeChannel(uuid.FromStringOrNil(v), chID)
	}
	for _, v := range req.Off {
		model.UnsubscribeChannel(uuid.FromStringOrNil(v), chID)
	}

	return c.NoContent(http.StatusNoContent)
}

// PostDeviceToken POST /notification/device のハンドラ
func PostDeviceToken(c echo.Context) error {
	user := c.Get("user").(*model.User)

	var req struct {
		Token string `json:"token" validate:"required"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	if _, err := model.RegisterDevice(user.GetUID(), req.Token); err != nil {
		c.Logger().Error(err)
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
		ch, err := model.GetChannelWithUserID(userID, v)
		if err != nil {
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get channels")
		}

		childIDs, err := model.GetChildrenChannelIDsWithUserID(userID, ch.GetCID().String())
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
