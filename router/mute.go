package router

import (
	"github.com/labstack/echo"
	"net/http"
)

// GetMutedChannelIDs GET /users/me/mute
func (h *Handlers) GetMutedChannelIDs(c echo.Context) error {
	uid := getRequestUserID(c)

	ids, err := h.Repo.GetMutedChannelIDs(uid)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, ids)
}

// PostMutedChannel POST /users/me/mute/:channelID
func (h *Handlers) PostMutedChannel(c echo.Context) error {
	uid := getRequestUserID(c)
	cid := getRequestParamAsUUID(c, paramChannelID)

	// ユーザーからチャンネルが見えるかどうか
	if ok, err := h.Repo.IsChannelAccessibleToUser(uid, cid); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	} else if !ok {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	// 強制通知チャンネルを確認
	if ch, err := h.Repo.GetChannel(cid); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	} else if ch.IsForced {
		return echo.NewHTTPError(http.StatusForbidden)
	}

	if err := h.Repo.MuteChannel(uid, cid); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

// DeleteMutedChannel DELETE /users/me/mute/:channelID
func (h *Handlers) DeleteMutedChannel(c echo.Context) error {
	uid := getRequestUserID(c)
	cid := getRequestParamAsUUID(c, paramChannelID)

	if err := h.Repo.UnmuteChannel(uid, cid); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}
