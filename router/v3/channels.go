package v3

import (
	vd "github.com/go-ozzo/ozzo-validation"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/realtime/viewer"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"gopkg.in/guregu/null.v3"
	"net/http"
)

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
