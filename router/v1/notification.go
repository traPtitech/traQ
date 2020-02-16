package v1

import (
	vd "github.com/go-ozzo/ozzo-validation"
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/utils/set"
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

// PutChannelSubscribersRequest PUT /channels/:channelID/notification リクエストボディ
type PutChannelSubscribersRequest struct {
	On  set.UUIDSet `json:"on"`
	Off set.UUIDSet `json:"off"`
}

// PutChannelSubscribers PUT /channels/:channelID/notification
func (h *Handlers) PutChannelSubscribers(c echo.Context) error {
	ch := getChannelFromContext(c)

	// プライベートチャンネルの通知は変更できない。
	if !ch.IsPublic {
		return herror.Forbidden("private channel's notification is not configurable")
	}

	var req PutChannelSubscribersRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	args := repository.ChangeChannelSubscriptionArgs{
		UpdaterID:    getRequestUserID(c),
		Subscription: map[uuid.UUID]bool{},
	}

	for _, id := range req.On.Array() {
		args.Subscription[id] = true
	}
	for _, id := range req.Off.Array() {
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

// PostDeviceTokenRequest POST /notification/device リクエストボディ
type PostDeviceTokenRequest struct {
	Token string `json:"token"`
}

func (r PostDeviceTokenRequest) Validate() error {
	return vd.ValidateStruct(&r,
		vd.Field(&r.Token, vd.Required, vd.RuneLength(1, 190)),
	)
}

// PostDeviceToken POST /notification/device
func (h *Handlers) PostDeviceToken(c echo.Context) error {
	userID := getRequestUserID(c)

	var req PostDeviceTokenRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
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
