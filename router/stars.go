package router

import (
	"github.com/traPtitech/traQ/event"
	"net/http"

	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/model"
)

// GetStars GET /users/me/stars
func GetStars(c echo.Context) error {
	me := c.Get("user").(*model.User)

	stars, err := model.GetStaredChannels(me.GetUID())
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	res := make([]string, len(stars))
	for i, v := range stars {
		res[i] = v.ID
	}

	return c.JSON(http.StatusOK, res)
}

// PutStars PUT /users/me/stars/:channelID
func PutStars(c echo.Context) error {
	user := c.Get("user").(*model.User)

	ch, err := validateChannelID(c.Param("channelID"), user.ID)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	if err := model.AddStar(user.GetUID(), ch.GetCID()); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	go event.Emit(event.ChannelStared, &event.UserChannelEvent{UserID: user.GetUID(), ChannelID: ch.GetCID()})
	return c.NoContent(http.StatusNoContent)
}

// DeleteStars DELETE /users/me/stars/:channelID
func DeleteStars(c echo.Context) error {
	user := c.Get("user").(*model.User)

	ch, err := validateChannelID(c.Param("channelID"), user.ID)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	if err := model.RemoveStar(user.GetUID(), ch.GetCID()); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	go event.Emit(event.ChannelUnstared, &event.UserChannelEvent{UserID: user.GetUID(), ChannelID: ch.GetCID()})
	return c.NoContent(http.StatusNoContent)
}
