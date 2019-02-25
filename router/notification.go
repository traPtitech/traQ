package router

import (
	"net/http"

	"github.com/labstack/echo"
	"github.com/satori/go.uuid"
)

// GetNotificationStatus GET /channels/:channelID/notification
func (h *Handlers) GetNotificationStatus(c echo.Context) error {
	ch := getChannelFromContext(c)

	// プライベートチャンネルの通知は取得できない。
	if !ch.IsPublic {
		return c.NoContent(http.StatusForbidden)
	}

	users, err := h.Repo.GetSubscribingUserIDs(ch.ID)
	if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	return c.JSON(http.StatusOK, users)
}

// PutNotificationStatus PUT /channels/:channelID/notification
func (h *Handlers) PutNotificationStatus(c echo.Context) error {
	ch := getChannelFromContext(c)

	// プライベートチャンネルの通知は変更できない。
	if !ch.IsPublic {
		return c.NoContent(http.StatusForbidden)
	}

	var req struct {
		On  []uuid.UUID `json:"on"`
		Off []uuid.UUID `json:"off"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	for _, id := range req.On {
		if ok, err := h.Repo.UserExists(id); err != nil {
			c.Logger().Error(err)
			return c.NoContent(http.StatusInternalServerError)
		} else if ok {
			if err := h.Repo.SubscribeChannel(id, ch.ID); err != nil {
				c.Logger().Error(err)
				return c.NoContent(http.StatusInternalServerError)
			}
		}
	}
	for _, id := range req.Off {
		err := h.Repo.UnsubscribeChannel(id, ch.ID)
		if err != nil {
			c.Logger().Error(err)
			return c.NoContent(http.StatusInternalServerError)
		}
	}

	return c.NoContent(http.StatusNoContent)
}

// PostDeviceToken POST /notification/device
func (h *Handlers) PostDeviceToken(c echo.Context) error {
	userID := getRequestUserID(c)

	var req struct {
		Token string `json:"token" validate:"required"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	if _, err := h.Repo.RegisterDevice(userID, req.Token); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusCreated)
}

// GetNotificationChannels GET /users/:userID/notification
func (h *Handlers) GetNotificationChannels(c echo.Context) error {
	userID := getRequestParamAsUUID(c, paramUserID)

	channelIDs, err := h.Repo.GetSubscribedChannelIDs(userID)
	if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, channelIDs)
}

// GetMyNotificationChannels GET /users/me/notification
func (h *Handlers) GetMyNotificationChannels(c echo.Context) error {
	userID := getRequestUserID(c)

	channelIDs, err := h.Repo.GetSubscribedChannelIDs(userID)
	if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, channelIDs)
}
