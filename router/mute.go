package router

import (
	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"net/http"
)

// GetMutedChannelIDs GET /users/me/mute
func GetMutedChannelIDs(c echo.Context) error {
	uid := getRequestUserID(c)

	ids, err := model.GetMutedChannelIDs(uid)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, ids)
}

// PostMutedChannel POST /users/me/mute/:channelID
func PostMutedChannel(c echo.Context) error {
	uid := getRequestUserID(c)
	cid := getRequestParamAsUUID(c, paramChannelID)

	if err := model.MuteChannel(uid, cid); err != nil {
		switch err {
		case model.ErrNotFound: // 存在しないチャンネルか、見えないチャンネル
			return echo.NewHTTPError(http.StatusNotFound)
		case model.ErrForbidden: // 強制通知チャンネルはミュート不可能
			return echo.NewHTTPError(http.StatusForbidden)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	go event.Emit(event.ChannelMuted, &event.UserChannelEvent{UserID: uid, ChannelID: cid})
	return c.NoContent(http.StatusNoContent)
}

// DeleteMutedChannel DELETE /users/me/mute/:channelID
func DeleteMutedChannel(c echo.Context) error {
	uid := getRequestUserID(c)
	cid := getRequestParamAsUUID(c, paramChannelID)

	if err := model.UnmuteChannel(uid, cid); err != nil {
		switch err {
		case model.ErrNotFound: // 存在しないチャンネルか、見えないチャンネル
			return echo.NewHTTPError(http.StatusNotFound)
		case model.ErrForbidden: // 強制通知チャンネルはミュート不可能
			return echo.NewHTTPError(http.StatusForbidden)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	go event.Emit(event.ChannelUnmuted, &event.UserChannelEvent{UserID: uid, ChannelID: cid})
	return c.NoContent(http.StatusNoContent)
}
