package router

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/repository"
	"net/http"

	"github.com/labstack/echo"
)

// GetNotificationStatus GET /channels/:channelID/notification
func (h *Handlers) GetNotificationStatus(c echo.Context) error {
	ch := getChannelFromContext(c)

	// プライベートチャンネルの通知は取得できない。
	if !ch.IsPublic {
		return forbidden("private channel's notification is not configurable")
	}

	users, err := h.Repo.GetSubscribingUserIDs(ch.ID)
	if err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}
	return c.JSON(http.StatusOK, users)
}

// PutNotificationStatus PUT /channels/:channelID/notification
func (h *Handlers) PutNotificationStatus(c echo.Context) error {
	ch := getChannelFromContext(c)

	// プライベートチャンネルの通知は変更できない。
	if !ch.IsPublic {
		return forbidden("private channel's notification is not configurable")
	}

	var req struct {
		On  []uuid.UUID `json:"on"`
		Off []uuid.UUID `json:"off"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return badRequest(err)
	}

	for _, id := range req.On {
		if ok, err := h.Repo.UserExists(id); err != nil {
			return internalServerError(err, h.requestContextLogger(c))
		} else if ok {
			if err := h.Repo.SubscribeChannel(id, ch.ID); err != nil {
				return internalServerError(err, h.requestContextLogger(c))
			}
		}
	}
	for _, id := range req.Off {
		err := h.Repo.UnsubscribeChannel(id, ch.ID)
		if err != nil {
			return internalServerError(err, h.requestContextLogger(c))
		}
	}

	return c.NoContent(http.StatusNoContent)
}

// PostDeviceToken POST /notification/device
func (h *Handlers) PostDeviceToken(c echo.Context) error {
	userID := getRequestUserID(c)

	var req struct {
		Token string `json:"token"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return badRequest(err)
	}

	if _, err := h.Repo.RegisterDevice(userID, req.Token); err != nil {
		switch {
		case repository.IsArgError(err):
			return badRequest(err)
		default:
			return internalServerError(err, h.requestContextLogger(c))
		}
	}

	return c.NoContent(http.StatusCreated)
}

// GetNotificationChannels GET /users/:userID/notification
func (h *Handlers) GetNotificationChannels(c echo.Context) error {
	userID := getRequestParamAsUUID(c, paramUserID)

	channelIDs, err := h.Repo.GetSubscribedChannelIDs(userID)
	if err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}

	return c.JSON(http.StatusOK, channelIDs)
}

// GetMyNotificationChannels GET /users/me/notification
func (h *Handlers) GetMyNotificationChannels(c echo.Context) error {
	userID := getRequestUserID(c)

	channelIDs, err := h.Repo.GetSubscribedChannelIDs(userID)
	if err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}

	return c.JSON(http.StatusOK, channelIDs)
}
