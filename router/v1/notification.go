package v1

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"net/http"

	"github.com/labstack/echo/v4"
)

// GetChannelSubscribers GET /channels/:channelID/notification
func (h *Handlers) GetChannelSubscribers(c echo.Context) error {
	ch := getChannelFromContext(c)

	// プライベートチャンネルの通知は取得できない。
	if !ch.IsPublic {
		return herror.Forbidden("private channel's notification is not configurable")
	}

	users, err := h.Repo.GetSubscribingUserIDs(ch.ID)
	if err != nil {
		return herror.InternalServerError(err)
	}
	return c.JSON(http.StatusOK, users)
}

// PutChannelSubscribers PUT /channels/:channelID/notification
func (h *Handlers) PutChannelSubscribers(c echo.Context) error {
	ch := getChannelFromContext(c)

	// プライベートチャンネルの通知は変更できない。
	if !ch.IsPublic {
		return herror.Forbidden("private channel's notification is not configurable")
	}

	var req struct {
		On  []uuid.UUID `json:"on"`
		Off []uuid.UUID `json:"off"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return herror.BadRequest(err)
	}

	args := repository.ChangeChannelSubscriptionArgs{
		UpdaterID:    getRequestUserID(c),
		Subscription: map[uuid.UUID]bool{},
	}

	for _, id := range req.On {
		args.Subscription[id] = true
	}
	for _, id := range req.Off {
		if args.Subscription[id] {
			// On, Offどっちにもあるものは相殺
			delete(args.Subscription, id)
		} else {
			args.Subscription[id] = false
		}
	}

	if err := h.Repo.ChangeChannelSubscription(ch.ID, args); err != nil {
		return herror.InternalServerError(err)
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
		return herror.BadRequest(err)
	}

	if _, err := h.Repo.RegisterDevice(userID, req.Token); err != nil {
		switch {
		case repository.IsArgError(err):
			return herror.BadRequest(err)
		default:
			return herror.InternalServerError(err)
		}
	}

	return c.NoContent(http.StatusCreated)
}

// GetNotificationChannels GET /users/:userID/notification
func (h *Handlers) GetNotificationChannels(c echo.Context) error {
	userID := getRequestParamAsUUID(c, consts.ParamUserID)

	channelIDs, err := h.Repo.GetSubscribedChannelIDs(userID)
	if err != nil {
		return herror.InternalServerError(err)
	}

	return c.JSON(http.StatusOK, channelIDs)
}

// GetMyNotificationChannels GET /users/me/notification
func (h *Handlers) GetMyNotificationChannels(c echo.Context) error {
	userID := getRequestUserID(c)

	channelIDs, err := h.Repo.GetSubscribedChannelIDs(userID)
	if err != nil {
		return herror.InternalServerError(err)
	}

	return c.JSON(http.StatusOK, channelIDs)
}
