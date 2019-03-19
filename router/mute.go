package router

import (
	"github.com/labstack/echo"
	"go.uber.org/zap"
	"net/http"
)

// GetMutedChannelIDs GET /users/me/mute
func (h *Handlers) GetMutedChannelIDs(c echo.Context) error {
	uid := getRequestUserID(c)

	ids, err := h.Repo.GetMutedChannelIDs(uid)
	if err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, ids)
}

// PostMutedChannel POST /users/me/mute/:channelID
func (h *Handlers) PostMutedChannel(c echo.Context) error {
	uid := getRequestUserID(c)
	cid := getRequestParamAsUUID(c, paramChannelID)
	ch := getChannelFromContext(c)

	// 強制通知チャンネルを確認
	if ch.IsForced {
		return c.NoContent(http.StatusForbidden)
	}

	if err := h.Repo.MuteChannel(uid, cid); err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

// DeleteMutedChannel DELETE /users/me/mute/:channelID
func (h *Handlers) DeleteMutedChannel(c echo.Context) error {
	uid := getRequestUserID(c)
	cid := getRequestParamAsUUID(c, paramChannelID)

	if err := h.Repo.UnmuteChannel(uid, cid); err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}
