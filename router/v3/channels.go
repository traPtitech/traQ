package v3

import (
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/realtime/viewer"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
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
