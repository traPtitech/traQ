package router

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/realtime/viewer"
	"github.com/traPtitech/traQ/repository"
	"gopkg.in/guregu/null.v3"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

// PostChannel リクエストボディ用構造体
type PostChannel struct {
	Name   string    `json:"name" validate:"required"`
	Parent uuid.UUID `json:"parent"`
}

// GetChannels GET /channels
func (h *Handlers) GetChannels(c echo.Context) error {
	userID := getRequestUserID(c)

	channelList, err := h.Repo.GetChannelsByUserID(userID)
	if err != nil {
		return internalServerError(err, h.requestContextLogger(c))
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
				return internalServerError(err, h.requestContextLogger(c))
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

// PostChannels POST /channels
func (h *Handlers) PostChannels(c echo.Context) error {
	userID := getRequestUserID(c)

	req := PostChannel{}
	if err := bindAndValidate(c, &req); err != nil {
		return badRequest(err)
	}

	// 親チャンネルがユーザーから見えないと作成できない
	if req.Parent != uuid.Nil {
		if ok, err := h.Repo.IsChannelAccessibleToUser(userID, req.Parent); err != nil {
			return internalServerError(err, h.requestContextLogger(c))
		} else if !ok {
			return badRequest("the parent channel doesn't exist")
		}
	}

	ch, err := h.Repo.CreatePublicChannel(req.Name, req.Parent, userID)
	if err != nil {
		switch {
		case repository.IsArgError(err):
			return badRequest(err)
		case err == repository.ErrAlreadyExists:
			return conflict("channel name conflicts")
		case err == repository.ErrChannelDepthLimitation:
			return badRequest("channel depth limit exceeded")
		case err == repository.ErrForbidden:
			return forbidden("invalid parent channel")
		default:
			return internalServerError(err, h.requestContextLogger(c))
		}
	}

	formatted, err := h.formatChannel(ch)
	if err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}
	return c.JSON(http.StatusCreated, formatted)
}

// GetChannelByChannelID GET /channels/:channelID
func (h *Handlers) GetChannelByChannelID(c echo.Context) error {
	ch := getChannelFromContext(c)

	formatted, err := h.formatChannel(ch)
	if err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}
	return c.JSON(http.StatusOK, formatted)
}

// PatchChannelByChannelID PATCH /channels/:channelID
func (h *Handlers) PatchChannelByChannelID(c echo.Context) error {
	channelID := getRequestParamAsUUID(c, paramChannelID)

	var req struct {
		Name       null.String `json:"name"`
		Visibility null.Bool   `json:"visibility"`
		Force      null.Bool   `json:"force"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return badRequest(err)
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
			return badRequest(err)
		case err == repository.ErrAlreadyExists:
			return conflict("channel name conflicts")
		case err == repository.ErrForbidden:
			return forbidden("the channel's name cannot be changed")
		default:
			return internalServerError(err, h.requestContextLogger(c))
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
		return badRequest(err)
	}

	// 子チャンネル作成
	ch, err := h.Repo.CreateChildChannel(req.Name, parentCh.ID, userID)
	if err != nil {
		switch {
		case repository.IsArgError(err):
			return badRequest(err)
		case err == repository.ErrAlreadyExists:
			return conflict("channel name conflicts")
		case err == repository.ErrChannelDepthLimitation:
			return badRequest("channel depth limit exceeded")
		case err == repository.ErrForbidden:
			return forbidden("invalid parent channel")
		default:
			return internalServerError(err, h.requestContextLogger(c))
		}
	}

	formatted, err := h.formatChannel(ch)
	if err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}
	return c.JSON(http.StatusCreated, formatted)
}

// PutChannelParent PUT /channels/:channelID/parent
func (h *Handlers) PutChannelParent(c echo.Context) error {
	channelID := getRequestParamAsUUID(c, paramChannelID)

	var req struct {
		Parent uuid.UUID `json:"parent"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return badRequest(err)
	}

	if err := h.Repo.ChangeChannelParent(channelID, req.Parent, getRequestUserID(c)); err != nil {
		switch err {
		case repository.ErrAlreadyExists:
			return conflict("channel name conflicts")
		case repository.ErrChannelDepthLimitation:
			return badRequest("channel depth limit exceeded")
		case repository.ErrForbidden:
			return forbidden("invalid parent channel")
		default:
			return internalServerError(err, h.requestContextLogger(c))
		}
	}

	return c.NoContent(http.StatusNoContent)
}

// DeleteChannelByChannelID DELETE /channels/:channelID
func (h *Handlers) DeleteChannelByChannelID(c echo.Context) error {
	channelID := getRequestParamAsUUID(c, paramChannelID)

	if err := h.Repo.DeleteChannel(channelID); err != nil {
		return internalServerError(err, h.requestContextLogger(c))
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
	channelID := getRequestParamAsUUID(c, paramChannelID)

	var req struct {
		Text string `json:"text"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return badRequest(err)
	}

	if err := h.Repo.UpdateChannel(channelID, repository.UpdateChannelArgs{
		UpdaterID: getRequestUserID(c),
		Topic:     null.StringFrom(req.Text),
	}); err != nil {
		return internalServerError(err, h.requestContextLogger(c))
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
	channelID := getRequestParamAsUUID(c, paramChannelID)

	var req channelEventsQuery
	if err := req.bind(c); err != nil {
		return badRequest(err)
	}

	if req.Limit > 200 || req.Limit == 0 {
		req.Limit = 200 // １度に取れるのは200件まで
	}

	events, more, err := h.Repo.GetChannelEvents(req.convert(channelID))
	if err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}
	c.Response().Header().Set(headerMore, strconv.FormatBool(more))
	return c.JSON(http.StatusOK, events)
}

// GetChannelStats GET /channels/:channelID/stats
func (h *Handlers) GetChannelStats(c echo.Context) error {
	channelID := getRequestParamAsUUID(c, paramChannelID)

	stats, err := h.Repo.GetChannelStats(channelID)
	if err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}
	return c.JSON(http.StatusOK, stats)
}

// GetChannelViewers GET /channels/:channelID/viewers
func (h *Handlers) GetChannelViewers(c echo.Context) error {
	channelID := getRequestParamAsUUID(c, paramChannelID)

	cv := h.Realtime.ViewerManager.GetChannelViewers(channelID)
	return c.JSON(http.StatusOK, viewer.ConvertToArray(cv))
}
