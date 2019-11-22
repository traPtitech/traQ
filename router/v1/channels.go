package v1

import (
	vd "github.com/go-ozzo/ozzo-validation"
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/realtime/viewer"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/utils/validator"
	"gopkg.in/guregu/null.v3"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

// GetChannels GET /channels
func (h *Handlers) GetChannels(c echo.Context) error {
	userID := getRequestUserID(c)

	channelList, err := h.Repo.GetChannelsByUserID(userID)
	if err != nil {
		return herror.InternalServerError(err)
	}

	chMap := make(map[string]*channelResponse, len(channelList))
	for _, ch := range channelList {
		entry, ok := chMap[ch.ID.String()]
		if !ok {
			entry = &channelResponse{}
			chMap[ch.ID.String()] = entry
		}

		entry.ChannelID = ch.ID.String()
		entry.Name = ch.Name
		entry.Visibility = ch.IsVisible
		entry.Topic = ch.Topic
		entry.Force = ch.IsForced
		entry.Private = !ch.IsPublic
		entry.DM = ch.IsDMChannel()

		if !ch.IsPublic {
			// プライベートチャンネルのメンバー取得
			member, err := h.Repo.GetPrivateChannelMemberIDs(ch.ID)
			if err != nil {
				return herror.InternalServerError(err)
			}
			entry.Member = member
		}

		if ch.ParentID != uuid.Nil {
			entry.Parent = ch.ParentID.String()
			parent, ok := chMap[ch.ParentID.String()]
			if !ok {
				parent = &channelResponse{
					ChannelID: ch.ParentID.String(),
				}
				chMap[ch.ParentID.String()] = parent
			}
			parent.Children = append(parent.Children, ch.ID)
		} else {
			parent, ok := chMap[""]
			if !ok {
				parent = &channelResponse{}
				chMap[""] = parent
			}
			parent.Children = append(parent.Children, ch.ID)
		}
	}

	res := make([]*channelResponse, 0, len(chMap))
	for _, v := range chMap {
		res = append(res, v)
	}
	return c.JSON(http.StatusOK, res)
}

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

	// 親チャンネルがユーザーから見えないと作成できない
	if req.Parent != uuid.Nil {
		if ok, err := h.Repo.IsChannelAccessibleToUser(userID, req.Parent); err != nil {
			return herror.InternalServerError(err)
		} else if !ok {
			return herror.BadRequest("the parent channel doesn't exist")
		}
	}

	ch, err := h.Repo.CreatePublicChannel(req.Name, req.Parent, userID)
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

// PatchChannelByChannelID PATCH /channels/:channelID
func (h *Handlers) PatchChannelByChannelID(c echo.Context) error {
	channelID := getRequestParamAsUUID(c, consts.ParamChannelID)

	var req struct {
		Name       null.String `json:"name"`
		Visibility null.Bool   `json:"visibility"`
		Force      null.Bool   `json:"force"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	args := repository.UpdateChannelArgs{
		UpdaterID:          getRequestUserID(c),
		Name:               req.Name,
		Visibility:         req.Visibility,
		ForcedNotification: req.Force,
	}
	if err := h.Repo.UpdateChannel(channelID, args); err != nil {
		switch {
		case repository.IsArgError(err):
			return herror.BadRequest(err)
		case err == repository.ErrAlreadyExists:
			return herror.Conflict("channel name conflicts")
		case err == repository.ErrForbidden:
			return herror.Forbidden("the channel's name cannot be changed")
		default:
			return herror.InternalServerError(err)
		}
	}
	return c.NoContent(http.StatusNoContent)
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
	ch, err := h.Repo.CreateChildChannel(req.Name, parentCh.ID, userID)
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

	formatted, err := h.formatChannel(ch)
	if err != nil {
		return herror.InternalServerError(err)
	}
	return c.JSON(http.StatusCreated, formatted)
}

// PutChannelParent PUT /channels/:channelID/parent
func (h *Handlers) PutChannelParent(c echo.Context) error {
	channelID := getRequestParamAsUUID(c, consts.ParamChannelID)

	var req struct {
		Parent uuid.UUID `json:"parent"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	if err := h.Repo.ChangeChannelParent(channelID, req.Parent, getRequestUserID(c)); err != nil {
		switch err {
		case repository.ErrAlreadyExists:
			return herror.Conflict("channel name conflicts")
		case repository.ErrChannelDepthLimitation:
			return herror.BadRequest("channel depth limit exceeded")
		case repository.ErrForbidden:
			return herror.Forbidden("invalid parent channel")
		default:
			return herror.InternalServerError(err)
		}
	}

	return c.NoContent(http.StatusNoContent)
}

// DeleteChannelByChannelID DELETE /channels/:channelID
func (h *Handlers) DeleteChannelByChannelID(c echo.Context) error {
	channelID := getRequestParamAsUUID(c, consts.ParamChannelID)

	if err := h.Repo.DeleteChannel(channelID); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
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
	channelID := getRequestParamAsUUID(c, consts.ParamChannelID)

	var req struct {
		Text string `json:"text"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	if err := h.Repo.UpdateChannel(channelID, repository.UpdateChannelArgs{
		UpdaterID: getRequestUserID(c),
		Topic:     null.StringFrom(req.Text),
	}); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

type channelEventsQuery struct {
	Limit     int        `query:"limit"`
	Offset    int        `query:"offset"`
	Since     *time.Time `query:"since"`
	Until     *time.Time `query:"until"`
	Inclusive bool       `query:"inclusive"`
	Order     string     `query:"order"`
}

func (q *channelEventsQuery) bind(c echo.Context) error {
	return bindAndValidate(c, q)
}

func (q *channelEventsQuery) convert(cid uuid.UUID) repository.ChannelEventsQuery {
	return repository.ChannelEventsQuery{
		Since:     null.TimeFromPtr(q.Since),
		Until:     null.TimeFromPtr(q.Until),
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

	cv := h.Realtime.ViewerManager.GetChannelViewers(channelID)
	return c.JSON(http.StatusOK, viewer.ConvertToArray(cv))
}
