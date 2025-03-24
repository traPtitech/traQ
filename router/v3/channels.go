package v3

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	vd "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/service/channel"
	channelService "github.com/traPtitech/traQ/service/channel"
	"github.com/traPtitech/traQ/service/viewer"
	"github.com/traPtitech/traQ/utils/optional"
	"github.com/traPtitech/traQ/utils/set"
	"github.com/traPtitech/traQ/utils/validator"
)

// GetChannels GET /channels
func (h *Handlers) GetChannels(c echo.Context) error {
	if isTrue(c.QueryParam("include-dm")) && len(c.QueryParam("path")) > 0 {
		return herror.BadRequest("include-dm and path cannot be specified at the same time")
	}

	var res echo.Map

	if channelPath := c.QueryParam("path"); channelPath != "" {
		channel, err := h.ChannelManager.GetChannelFromPath(channelPath)
		if err != nil {
			if errors.Is(err, channelService.ErrInvalidChannelPath) {
				return herror.HTTPError(http.StatusNotFound, err)
			}
			return herror.InternalServerError(err)
		}
		res = echo.Map{
			"public": []*Channel{
				formatChannel(channel, h.ChannelManager.PublicChannelTree().GetChildrenIDs(channel.ID)),
			},
		}
		return extension.ServeJSONWithETag(c, res)
	}

	res = echo.Map{
		"public": h.ChannelManager.PublicChannelTree(),
	}
	if isTrue(c.QueryParam("include-dm")) {
		mapping, err := h.ChannelManager.GetDMChannelMapping(getRequestUserID(c))
		if err != nil {
			return herror.InternalServerError(err)
		}
		res["dm"] = formatDMChannels(mapping)
	}
	return extension.ServeJSONWithETag(c, res)
}

// PostChannelRequest POST /channels リクエストボディ
type PostChannelRequest struct {
	Name   string                 `json:"name"`
	Parent optional.Of[uuid.UUID] `json:"parent"`
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

	ch, err := h.ChannelManager.CreatePublicChannel(req.Name, req.Parent.V, userID)
	if err != nil {
		switch err {
		case channel.ErrChannelArchived:
			return herror.BadRequest("parent channel has been archived")
		case channel.ErrInvalidChannelName:
			return herror.BadRequest("invalid channel name")
		case channel.ErrInvalidParentChannel:
			return herror.BadRequest("invalid parent channel")
		case channel.ErrTooDeepChannel:
			return herror.BadRequest("channel depth limit exceeded")
		case channel.ErrChannelNameConflicts:
			return herror.Conflict("channel name conflicts")
		default:
			return herror.InternalServerError(err)
		}
	}

	return c.JSON(http.StatusCreated, formatChannel(ch, make([]uuid.UUID, 0)))
}

// GetChannel GET /channels/:channelID
func (h *Handlers) GetChannel(c echo.Context) error {
	ch := getParamChannel(c)
	return c.JSON(http.StatusOK, formatChannel(ch, h.ChannelManager.PublicChannelTree().GetChildrenIDs(ch.ID)))
}

// PatchChannelRequest PATCH /channels/:channelID リクエストボディ
type PatchChannelRequest struct {
	Name     optional.Of[string]    `json:"name"`
	Archived optional.Of[bool]      `json:"archived"`
	Force    optional.Of[bool]      `json:"force"`
	Parent   optional.Of[uuid.UUID] `json:"parent"`
}

func (r PatchChannelRequest) Validate() error {
	return vd.ValidateStruct(&r,
		vd.Field(&r.Name, append(validator.ChannelNameRule, validator.RequiredIfValid)...),
	)
}

// EditChannel PATCH /channels/:channelID
func (h *Handlers) EditChannel(c echo.Context) error {
	channelID := getParamAsUUID(c, consts.ParamChannelID)

	var req PatchChannelRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	if req.Archived.Valid {
		if req.Archived.V {
			if err := h.ChannelManager.ArchiveChannel(channelID, getRequestUserID(c)); err != nil {
				switch err {
				case channel.ErrInvalidChannel:
					return herror.BadRequest("invalid channel id")
				default:
					return herror.InternalServerError(err)
				}
			}
		} else {
			if err := h.ChannelManager.UnarchiveChannel(channelID, getRequestUserID(c)); err != nil {
				switch err {
				case channel.ErrInvalidParentChannel:
					return herror.BadRequest("the parent channel has been archived")
				default:
					return herror.InternalServerError(err)
				}
			}
		}
	}

	args := repository.UpdateChannelArgs{
		UpdaterID:          getRequestUserID(c),
		Name:               req.Name,
		ForcedNotification: req.Force,
		Parent:             req.Parent,
	}
	if err := h.ChannelManager.UpdateChannel(channelID, args); err != nil {
		switch err {
		case channel.ErrInvalidChannelName:
			return herror.BadRequest("invalid channel name")
		case channel.ErrInvalidParentChannel:
			return herror.BadRequest("invalid parent channel")
		case channel.ErrTooDeepChannel:
			return herror.BadRequest("channel depth limit exceeded")
		case channel.ErrChannelNameConflicts:
			return herror.Conflict("channel name conflicts")
		default:
			return herror.InternalServerError(err)
		}
	}
	return c.NoContent(http.StatusNoContent)
}

// GetChannelViewers GET /channels/:channelID/viewers
func (h *Handlers) GetChannelViewers(c echo.Context) error {
	channelID := getParamAsUUID(c, consts.ParamChannelID)
	cv := h.VM.GetChannelViewers(channelID)
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
		vd.Field(&r.Topic, vd.RuneLength(0, 500)),
	)
}

// EditChannelTopic PUT /channels/:channelID/topic
func (h *Handlers) EditChannelTopic(c echo.Context) error {
	ch := getParamChannel(c)

	var req PutChannelTopicRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	if err := h.ChannelManager.UpdateChannel(ch.ID, repository.UpdateChannelArgs{
		UpdaterID: getRequestUserID(c),
		Topic:     optional.From(req.Topic),
	}); err != nil {
		switch err {
		case channel.ErrChannelArchived:
			return herror.BadRequest("channel has been archived")
		default:
			return herror.InternalServerError(err)
		}
	}

	return c.NoContent(http.StatusNoContent)
}

// GetChannelPins GET /channels/:channelID/pins
func (h *Handlers) GetChannelPins(c echo.Context) error {
	channelID := getParamAsUUID(c, consts.ParamChannelID)

	pins, err := h.Repo.GetPinnedMessageByChannelID(channelID)
	if err != nil {
		return herror.InternalServerError(err)
	}

	return c.JSON(http.StatusOK, formatPins(pins))
}

type channelEventsQuery struct {
	Limit     int                    `query:"limit"`
	Offset    int                    `query:"offset"`
	Since     optional.Of[time.Time] `query:"since"`
	Until     optional.Of[time.Time] `query:"until"`
	Inclusive bool                   `query:"inclusive"`
	Order     string                 `query:"order"`
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
	if err := bindAndValidate(c, &req); err != nil {
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
	On set.UUID `json:"on"`
}

// SetChannelSubscribers PUT /channels/:channelID/subscribers
func (h *Handlers) SetChannelSubscribers(c echo.Context) error {
	ch := getParamChannel(c)

	var req PutChannelSubscribersRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	subscriptions, err := h.Repo.GetChannelSubscriptions(repository.ChannelSubscriptionQuery{}.SetChannel(ch.ID).SetLevel(model.ChannelSubscribeLevelMarkAndNotify))
	if err != nil {
		return herror.InternalServerError(err)
	}

	subs := map[uuid.UUID]model.ChannelSubscribeLevel{}
	for _, subscription := range subscriptions {
		subs[subscription.UserID] = model.ChannelSubscribeLevelNone
	}
	for _, id := range req.On.Array() {
		subs[id] = model.ChannelSubscribeLevelMarkAndNotify
	}

	if err := h.ChannelManager.ChangeChannelSubscriptions(ch.ID, subs, true, getRequestUserID(c)); err != nil {
		switch err {
		case channel.ErrInvalidChannel:
			return herror.Forbidden("the channel's subscriptions is not configurable")
		case channel.ErrForcedNotification:
			return herror.Forbidden("the channel's subscriptions is not configurable")
		default:
			return herror.InternalServerError(err)
		}
	}
	return c.NoContent(http.StatusNoContent)
}

// PatchChannelSubscribersRequest PATCH /channels/:channelID/subscribers リクエストボディ
type PatchChannelSubscribersRequest struct {
	On  set.UUID `json:"on"`
	Off set.UUID `json:"off"`
}

// EditChannelSubscribers PATCH /channels/:channelID/subscribers
func (h *Handlers) EditChannelSubscribers(c echo.Context) error {
	ch := getParamChannel(c)

	var req PatchChannelSubscribersRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	subscriptions := map[uuid.UUID]model.ChannelSubscribeLevel{}
	for _, id := range req.On.Array() {
		subscriptions[id] = model.ChannelSubscribeLevelMarkAndNotify
	}
	for _, id := range req.Off.Array() {
		if _, ok := subscriptions[id]; ok {
			// On, Offどっちにもあるものは相殺
			delete(subscriptions, id)
		} else {
			subscriptions[id] = model.ChannelSubscribeLevelNone
		}
	}

	if err := h.ChannelManager.ChangeChannelSubscriptions(ch.ID, subscriptions, true, getRequestUserID(c)); err != nil {
		switch err {
		case channel.ErrInvalidChannel:
			return herror.Forbidden("the channel's subscriptions is not configurable")
		case channel.ErrForcedNotification:
			return herror.Forbidden("the channel's subscriptions is not configurable")
		default:
			return herror.InternalServerError(err)
		}
	}
	return c.NoContent(http.StatusNoContent)
}

// GetUserDMChannel GET /users/:userID/dm-channel
func (h *Handlers) GetUserDMChannel(c echo.Context) error {
	userID := getParamAsUUID(c, consts.ParamUserID)
	myID := getRequestUserID(c)

	// DMチャンネルを取得
	ch, err := h.ChannelManager.GetDMChannel(myID, userID)
	if err != nil {
		return herror.InternalServerError(err)
	}

	return c.JSON(http.StatusOK, &DMChannel{ID: ch.ID, UserID: userID})
}

// GetChannelPath GET /channels/:channelID/path
func (h *Handlers) GetChannelPath(c echo.Context) error {
	channelID := getParamAsUUID(c, consts.ParamChannelID)

	channelPath := h.ChannelManager.GetChannelPathFromID(channelID)

	return c.JSON(http.StatusOK, echo.Map{"path": channelPath})
}
