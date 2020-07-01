package v1

import (
	vd "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/service/channel"
	"github.com/traPtitech/traQ/service/viewer"
	"github.com/traPtitech/traQ/utils/optional"
	"github.com/traPtitech/traQ/utils/validator"
	"net/http"
	"strconv"
	"strings"
)

// PostChannelRequest POST /channels リクエストボディ
type PostChannelRequest struct {
	Name   string    `json:"name"`
	Parent uuid.UUID `json:"parent"`
}

func (r PostChannelRequest) Validate() error {
	return vd.ValidateStruct(&r,
		vd.Field(&r.Name, validator.ChannelNameRuleRequired...),
	)
}

// PostChannels POST /channels
func (h *Handlers) PostChannels(c echo.Context) error {
	userID := getRequestUserID(c)

	var req PostChannelRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	ch, err := h.ChannelManager.CreatePublicChannel(req.Name, req.Parent, userID)
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

	formatted, err := h.formatChannel(ch)
	if err != nil {
		return herror.InternalServerError(err)
	}
	return c.JSON(http.StatusCreated, formatted)
}

// GetChannelByChannelID GET /channels/:channelID
func (h *Handlers) GetChannelByChannelID(c echo.Context) error {
	ch := getChannelFromContext(c)

	formatted, err := h.formatChannel(ch)
	if err != nil {
		return herror.InternalServerError(err)
	}
	return c.JSON(http.StatusOK, formatted)
}

// PostChannelChildren POST /channels/:channelID/children
func (h *Handlers) PostChannelChildren(c echo.Context) error {
	userID := getRequestUserID(c)
	parentCh := getChannelFromContext(c)

	var req struct {
		Name string `json:"name"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	// 子チャンネル作成
	ch, err := h.ChannelManager.CreatePublicChannel(req.Name, parentCh.ID, userID)
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

	formatted, err := h.formatChannel(ch)
	if err != nil {
		return herror.InternalServerError(err)
	}
	return c.JSON(http.StatusCreated, formatted)
}

// GetTopic GET /channels/:channelID/topic
func (h *Handlers) GetTopic(c echo.Context) error {
	ch := getChannelFromContext(c)
	return c.JSON(http.StatusOK, map[string]string{
		"text": ch.Topic,
	})
}

// PutTopic PUT /channels/:channelID/topic
func (h *Handlers) PutTopic(c echo.Context) error {
	ch := getChannelFromContext(c)

	var req struct {
		Text string `json:"text"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	if err := h.ChannelManager.UpdateChannel(ch.ID, repository.UpdateChannelArgs{
		UpdaterID: getRequestUserID(c),
		Topic:     optional.StringFrom(req.Text),
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

type channelEventsQuery struct {
	Limit     int           `query:"limit"`
	Offset    int           `query:"offset"`
	Since     optional.Time `query:"since"`
	Until     optional.Time `query:"until"`
	Inclusive bool          `query:"inclusive"`
	Order     string        `query:"order"`
}

func (q *channelEventsQuery) bind(c echo.Context) error {
	return bindAndValidate(c, q)
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
	channelID := getRequestParamAsUUID(c, consts.ParamChannelID)

	var req channelEventsQuery
	if err := req.bind(c); err != nil {
		return err
	}

	if req.Limit > 200 || req.Limit == 0 {
		req.Limit = 200 // １度に取れるのは200件まで
	}

	events, more, err := h.Repo.GetChannelEvents(req.convert(channelID))
	if err != nil {
		return herror.InternalServerError(err)
	}
	c.Response().Header().Set(consts.HeaderMore, strconv.FormatBool(more))
	return c.JSON(http.StatusOK, events)
}

// GetChannelStats GET /channels/:channelID/stats
func (h *Handlers) GetChannelStats(c echo.Context) error {
	channelID := getRequestParamAsUUID(c, consts.ParamChannelID)

	stats, err := h.Repo.GetChannelStats(channelID)
	if err != nil {
		return herror.InternalServerError(err)
	}
	return c.JSON(http.StatusOK, stats)
}

// GetChannelViewers GET /channels/:channelID/viewers
func (h *Handlers) GetChannelViewers(c echo.Context) error {
	channelID := getRequestParamAsUUID(c, consts.ParamChannelID)

	cv := h.VM.GetChannelViewers(channelID)
	return c.JSON(http.StatusOK, viewer.ConvertToArray(cv))
}
