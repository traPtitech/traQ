package v3

import (
	"fmt"
	vd "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/realtime/viewer"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/utils/set"
	"github.com/traPtitech/traQ/utils/validator"
	"gopkg.in/guregu/null.v3"
	"net/http"
	"strconv"
	"strings"
)

// GetChannels GET /channels
func (h *Handlers) GetChannels(c echo.Context) error {
	res := echo.Map{
		"public": h.Repo.GetChannelTree(),
	}

	if isTrue(c.QueryParam("include-dm")) {
		mapping, err := h.Repo.GetDirectMessageChannelMapping(getRequestUserID(c))
		if err != nil {
			return herror.InternalServerError(err)
		}
		res["dm"] = formatDMChannels(mapping)
	}

	return c.JSON(http.StatusOK, res)
}

// PostChannelRequest POST /channels リクエストボディ
type PostChannelRequest struct {
	Name   string        `json:"name"`
	Parent uuid.NullUUID `json:"parent"`
}

func (r PostChannelRequest) Validate() error {
	return vd.ValidateStruct(&r,
		vd.Field(&r.Name, validator.ChannelNameRuleRequired...),
	)
}

// CreateChannels POST /channels
func (h *Handlers) CreateChannels(c echo.Context) error {
	userID := getRequestUserID(c)

	var req PostChannelRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	ch, err := h.Repo.CreatePublicChannel(req.Name, req.Parent.UUID, userID)
	if err != nil {
		switch {
		case repository.IsArgError(err):
			return herror.BadRequest(err)
		case err == repository.ErrAlreadyExists:
			return herror.Conflict("channel name conflicts")
		case err == repository.ErrChannelDepthLimitation:
			return herror.BadRequest("channel depth limit exceeded")
		case err == repository.ErrForbidden:
			return herror.Forbidden("invalid parent channel")
		default:
			return herror.InternalServerError(err)
		}
	}

	return c.JSON(http.StatusCreated, formatChannel(ch, make([]uuid.UUID, 0)))
}

// GetChannel GET /channels/:channelID
func (h *Handlers) GetChannel(c echo.Context) error {
	ch := getParamChannel(c)
	return c.JSON(http.StatusOK, formatChannel(ch, h.Repo.GetChannelTree().GetChildrenIDs(ch.ID)))
}

// PatchChannelRequest PATCH /channels/:channelID リクエストボディ
type PatchChannelRequest struct {
	Name     null.String   `json:"name"`
	Archived null.Bool     `json:"archived"`
	Force    null.Bool     `json:"force"`
	Parent   uuid.NullUUID `json:"parent"`
}

func (r PatchChannelRequest) Validate() error {
	return vd.ValidateStruct(&r,
		vd.Field(&r.Name, validator.ChannelNameRule...),
	)
}

// EditChannel PATCH /channels/:channelID
func (h *Handlers) EditChannel(c echo.Context) error {
	channelID := getParamAsUUID(c, consts.ParamChannelID)

	var req PatchChannelRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	args := repository.UpdateChannelArgs{
		UpdaterID:          getRequestUserID(c),
		Name:               req.Name,
		Visibility:         null.NewBool(!req.Archived.Bool, req.Archived.Valid),
		ForcedNotification: req.Force,
		Parent:             req.Parent,
	}
	if err := h.Repo.UpdateChannel(channelID, args); err != nil {
		switch {
		case repository.IsArgError(err):
			return herror.BadRequest(err)
		case err == repository.ErrAlreadyExists:
			return herror.Conflict("channel name conflicts")
		case err == repository.ErrForbidden:
			return herror.Forbidden()
		case err == repository.ErrChannelDepthLimitation:
			return herror.BadRequest("channel depth limit exceeded")
		default:
			return herror.InternalServerError(err)
		}
	}
	return c.NoContent(http.StatusNoContent)
}

// GetChannelViewers GET /channels/:channelID/viewers
func (h *Handlers) GetChannelViewers(c echo.Context) error {
	channelID := getParamAsUUID(c, consts.ParamChannelID)
	cv := h.Realtime.ViewerManager.GetChannelViewers(channelID)
	return c.JSON(http.StatusOK, viewer.ConvertToArray(cv))
}

// GetChannelStats GET /channels/:channelID/stats
func (h *Handlers) GetChannelStats(c echo.Context) error {
	channelID := getParamAsUUID(c, consts.ParamChannelID)
	stats, err := h.Repo.GetChannelStats(channelID)
	if err != nil {
		return herror.InternalServerError(err)
	}
	return c.JSON(http.StatusOK, stats)
}

// GetChannelTopic GET /channels/:channelID/topic
func (h *Handlers) GetChannelTopic(c echo.Context) error {
	return c.JSON(http.StatusOK, echo.Map{"topic": getParamChannel(c).Topic})
}

// PutChannelTopicRequest PUT /channels/:channelID/topic リクエストボディ
type PutChannelTopicRequest struct {
	Topic string `json:"topic"`
}

func (r PutChannelTopicRequest) Validate() error {
	return vd.ValidateStruct(&r,
		vd.Field(&r.Topic, vd.RuneLength(0, 200)),
	)
}

// EditChannelTopic PUT /channels/:channelID/topic
func (h *Handlers) EditChannelTopic(c echo.Context) error {
	ch := getParamChannel(c)

	if ch.IsArchived() {
		return herror.BadRequest(fmt.Sprintf("channel #%s has been archived", h.Repo.GetChannelTree().GetChannelPath(ch.ID)))
	}

	var req PutChannelTopicRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	if err := h.Repo.UpdateChannel(ch.ID, repository.UpdateChannelArgs{
		UpdaterID: getRequestUserID(c),
		Topic:     null.StringFrom(req.Topic),
	}); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

// GetChannelPins GET /channels/:channelID/pins
func (h *Handlers) GetChannelPins(c echo.Context) error {
	channelID := getParamAsUUID(c, consts.ParamChannelID)

	pins, err := h.Repo.GetPinsByChannelID(channelID)
	if err != nil {
		return herror.InternalServerError(err)
	}

	return c.JSON(http.StatusOK, formatPins(pins))
}

type channelEventsQuery struct {
	Limit     int       `query:"limit"`
	Offset    int       `query:"offset"`
	Since     null.Time `query:"since"`
	Until     null.Time `query:"until"`
	Inclusive bool      `query:"inclusive"`
	Order     string    `query:"order"`
}

func (q *channelEventsQuery) bind(c echo.Context) error {
	return bindAndValidate(c, q)
}

func (q *channelEventsQuery) Validate() error {
	if q.Limit == 0 {
		q.Limit = 20
	}
	return vd.ValidateStruct(q,
		vd.Field(&q.Limit, vd.Min(1), vd.Max(200)),
		vd.Field(&q.Offset, vd.Min(0)),
	)
}

func (q *channelEventsQuery) convert(cid uuid.UUID) repository.ChannelEventsQuery {
	return repository.ChannelEventsQuery{
		Since:     q.Since,
		Until:     q.Until,
		Inclusive: q.Inclusive,
		Limit:     q.Limit,
		Offset:    q.Offset,
		Asc:       strings.ToLower(q.Order) == "asc",
		Channel:   cid,
	}
}

// GetChannelEvents GET /channels/:channelID/events
func (h *Handlers) GetChannelEvents(c echo.Context) error {
	channelID := getParamAsUUID(c, consts.ParamChannelID)

	var req channelEventsQuery
	if err := req.bind(c); err != nil {
		return err
	}

	events, more, err := h.Repo.GetChannelEvents(req.convert(channelID))
	if err != nil {
		return herror.InternalServerError(err)
	}
	c.Response().Header().Set(consts.HeaderMore, strconv.FormatBool(more))
	return c.JSON(http.StatusOK, events)
}

// GetChannelSubscribers GET /channels/:channelID/subscribers
func (h *Handlers) GetChannelSubscribers(c echo.Context) error {
	ch := getParamChannel(c)

	// プライベートチャンネル・強制通知チャンネルの設定は取得できない。
	if !ch.IsPublic || ch.IsForced {
		return herror.Forbidden()
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

// PutChannelSubscribersRequest PUT /channels/:channelID/subscribers リクエストボディ
type PutChannelSubscribersRequest struct {
	On set.UUIDSet `json:"on"`
}

// SetChannelSubscribers PUT /channels/:channelID/subscribers
func (h *Handlers) SetChannelSubscribers(c echo.Context) error {
	ch := getParamChannel(c)

	// プライベートチャンネル・強制通知チャンネルの設定は取得できない。
	if !ch.IsPublic || ch.IsForced {
		return herror.Forbidden()
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

	subscriptions, err := h.Repo.GetChannelSubscriptions(repository.ChannelSubscriptionQuery{}.SetChannel(ch.ID).SetLevel(model.ChannelSubscribeLevelMarkAndNotify))
	if err != nil {
		return herror.InternalServerError(err)
	}

	for _, subscription := range subscriptions {
		args.Subscription[subscription.UserID] = model.ChannelSubscribeLevelNone
	}
	for _, id := range req.On.Array() {
		args.Subscription[id] = model.ChannelSubscribeLevelMarkAndNotify
	}

	if err := h.Repo.ChangeChannelSubscription(ch.ID, args); err != nil {
		return herror.InternalServerError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// PatchChannelSubscribersRequest PATCH /channels/:channelID/subscribers リクエストボディ
type PatchChannelSubscribersRequest struct {
	On  set.UUIDSet `json:"on"`
	Off set.UUIDSet `json:"off"`
}

// EditChannelSubscribers PATCH /channels/:channelID/subscribers
func (h *Handlers) EditChannelSubscribers(c echo.Context) error {
	ch := getParamChannel(c)

	// プライベートチャンネル・強制通知チャンネルの設定は取得できない。
	if !ch.IsPublic || ch.IsForced {
		return herror.Forbidden()
	}

	var req PatchChannelSubscribersRequest
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

// GetUserDMChannel GET /users/:userID/dm-channel
func (h *Handlers) GetUserDMChannel(c echo.Context) error {
	userID := getParamAsUUID(c, consts.ParamUserID)
	myID := getRequestUserID(c)

	// DMチャンネルを取得
	ch, err := h.Repo.GetDirectMessageChannel(myID, userID)
	if err != nil {
		return herror.InternalServerError(err)
	}

	return c.JSON(http.StatusOK, &DMChannel{ID: ch.ID, UserID: userID})
}
