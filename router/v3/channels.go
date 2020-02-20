package v3

import (
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/realtime/viewer"
	"github.com/traPtitech/traQ/router/consts"
	"net/http"
)

// GetChannelViewers GET /channels/:channelID/viewers
func (h *Handlers) GetChannelViewers(c echo.Context) error {
	channelID := getParamAsUUID(c, consts.ParamChannelID)
	cv := h.Realtime.ViewerManager.GetChannelViewers(channelID)
	return c.JSON(http.StatusOK, viewer.ConvertToArray(cv))
}
