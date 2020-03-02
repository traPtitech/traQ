package v3

import (
	vd "github.com/go-ozzo/ozzo-validation"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/realtime/viewer"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/utils/validator"
	"gopkg.in/guregu/null.v3"
	"net/http"
)

// GetChannels GET /channels
func (h *Handlers) GetChannels(c echo.Context) error {
	channelList, err := h.Repo.GetChannelsByUserID(uuid.Nil)
	if err != nil {
		return herror.InternalServerError(err)
	}

	res := make([]*Channel, 0, len(channelList))
	chMap := make(map[uuid.UUID]*Channel, len(channelList))
	for _, ch := range channelList {
		entry, ok := chMap[ch.ID]
		if !ok {
			entry = &Channel{
				ID:       ch.ID,
				Children: make([]uuid.UUID, 0),
			}
			chMap[ch.ID] = entry
		}

		entry.Name = ch.Name
		entry.Topic = ch.Topic
		entry.Visibility = ch.IsVisible
		entry.Force = ch.IsForced
		if ch.ParentID != uuid.Nil {
			entry.ParentID = uuid.NullUUID{UUID: ch.ParentID, Valid: true}
			parent, ok := chMap[ch.ParentID]
			if !ok {
				parent = &Channel{
					ID:       ch.ParentID,
					Children: make([]uuid.UUID, 0),
				}
				chMap[ch.ParentID] = parent
			}
			parent.Children = append(parent.Children, ch.ID)
		}

		res = append(res, entry)
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

	childrenID, err := h.Repo.GetChildrenChannelIDs(ch.ID)
	if err != nil {
		return herror.InternalServerError(err)
	}

	return c.JSON(http.StatusOK, formatChannel(ch, childrenID))
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
	var req PutChannelTopicRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	channelID := getParamAsUUID(c, consts.ParamChannelID)
	if err := h.Repo.UpdateChannel(channelID, repository.UpdateChannelArgs{
		UpdaterID: getRequestUserID(c),
		Topic:     null.StringFrom(req.Topic),
	}); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}
