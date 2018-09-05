package router

import (
	"github.com/labstack/echo"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/model"
	"net/http"
)

// GetMutedChannelIDs GET /users/me/mute
func GetMutedChannelIDs(c echo.Context) error {
	uid := c.Get("user").(*model.User).GetUID()

	ids, err := model.GetMutedChannelIDs(uid)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, ids)
}

// PostMutedChannel POST /users/me/mute/:channelID
func PostMutedChannel(c echo.Context) error {
	uid := c.Get("user").(*model.User).GetUID()
	cid := uuid.FromStringOrNil(c.Param("channelID"))

	if err := model.MuteChannel(uid, cid); err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound)
		case model.ErrForbidden:
			return echo.NewHTTPError(http.StatusForbidden)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	return c.NoContent(http.StatusNoContent)
}

// DeleteMutedChannel DELETE /users/me/mute/:channelID
func DeleteMutedChannel(c echo.Context) error {
	uid := c.Get("user").(*model.User).GetUID()
	cid := uuid.FromStringOrNil(c.Param("channelID"))

	if err := model.UnmuteChannel(uid, cid); err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound)
		case model.ErrForbidden:
			return echo.NewHTTPError(http.StatusForbidden)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	return c.NoContent(http.StatusNoContent)
}
