package v1

import (
	vd "github.com/go-ozzo/ozzo-validation"
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
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

	subscriptions, err := h.Repo.GetChannelSubscriptions(repository.ChannelSubscriptionQuery{}.SetChannel(ch.ID).SetLevel(model.ChannelSubscribeLevelMarkAndNotify))
	if err != nil {
		return herror.InternalServerError(err)
	}
	result := make([]uuid.UUID, 0)
	for _, subscription := range subscriptions {
		result = append(result, subscription.UserID)
	}

	return c.JSON(http.StatusOK, result)
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
		Subscription: map[uuid.UUID]model.ChannelSubscribeLevel{},
		KeepOffLevel: true,
	}

	for _, id := range req.On.Array() {
		args.Subscription[id] = model.ChannelSubscribeLevelMarkAndNotify
	}
	for _, id := range req.Off.Array() {
		if _, ok := args.Subscription[id]; ok {
			// On, Offどっちにもあるものは相殺
			delete(args.Subscription, id)
		} else {
			args.Subscription[id] = model.ChannelSubscribeLevelNone
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
	return h.getUserNotificationChannels(c, getRequestParamAsUUID(c, consts.ParamUserID))
}

// GetMyNotificationChannels GET /users/me/notification
func (h *Handlers) GetMyNotificationChannels(c echo.Context) error {
	return h.getUserNotificationChannels(c, getRequestUserID(c))
}

func (h *Handlers) getUserNotificationChannels(c echo.Context, userID uuid.UUID) error {
	subscriptions, err := h.Repo.GetChannelSubscriptions(repository.ChannelSubscriptionQuery{}.SetUser(userID).SetLevel(model.ChannelSubscribeLevelMarkAndNotify))
	if err != nil {
		return herror.InternalServerError(err)
	}
	result := make([]uuid.UUID, 0)
	for _, subscription := range subscriptions {
		result = append(result, subscription.ChannelID)
	}
	return c.JSON(http.StatusOK, result)
}
