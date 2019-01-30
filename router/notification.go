package router

import (
	"net/http"

	"github.com/labstack/echo"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/model"
)

// GetNotificationStatus GET /channels/:channelID/notification
func GetNotificationStatus(c echo.Context) error {
	userID := getRequestUserID(c)
	channelID := getRequestParamAsUUID(c, paramChannelID)

	ch, err := validateChannelID(channelID, userID)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound, "this channel is not found")
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to find the specified channel")
		}
	}

	// プライベートチャンネルの通知は取得できない。
	if !ch.IsPublic {
		return echo.NewHTTPError(http.StatusForbidden)
	}

	users, err := model.GetSubscribingUser(ch.ID)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	result := make([]string, len(users))
	for i, v := range users {
		result[i] = v.String()
	}

	return c.JSON(http.StatusOK, result)
}

// PutNotificationStatus PUT /channels/:channelID/notification
func PutNotificationStatus(c echo.Context) error {
	userID := getRequestUserID(c)
	channelID := getRequestParamAsUUID(c, paramChannelID)

	ch, err := validateChannelID(channelID, userID)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound, "this channel is not found")
		default:
			c.Logger().Error(err)
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

	for _, v := range req.On {
		err := model.SubscribeChannel(uuid.FromStringOrNil(v), ch.ID)
		if err != nil {
			switch err {
			case model.ErrNotFound:
				break
			default:
				c.Logger().Error(err)
				return echo.NewHTTPError(http.StatusInternalServerError)
			}
		}
	}
	for _, v := range req.Off {
		err := model.UnsubscribeChannel(uuid.FromStringOrNil(v), ch.ID)
		if err != nil {
			switch err {
			case model.ErrNotFound:
				break
			default:
				c.Logger().Error(err)
				return echo.NewHTTPError(http.StatusInternalServerError)
			}
		}
	}

	return c.NoContent(http.StatusNoContent)
}

// PostDeviceToken POST /notification/device
func PostDeviceToken(c echo.Context) error {
	userID := getRequestUserID(c)

	var req struct {
		Token string `json:"token" validate:"required"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	if _, err := model.RegisterDevice(userID, req.Token); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusCreated)
}

// GetNotificationChannels GET /users/:userID/notification
func GetNotificationChannels(c echo.Context) error {
	userID := getRequestParamAsUUID(c, paramUserID)

	if ok, err := model.UserExists(userID); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	} else if !ok {
		return echo.NewHTTPError(http.StatusNotFound)
	}

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

		res[i], err = formatChannel(ch)
		if err != nil {
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}
	return c.JSON(http.StatusOK, res)
}
