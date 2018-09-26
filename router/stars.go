package router

import (
	"github.com/traPtitech/traQ/event"
	"net/http"

	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/model"
)

// GetStars GET /users/me/stars
func GetStars(c echo.Context) error {
	userID := getRequestUserID(c)

	stars, err := model.GetStaredChannels(userID)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, stars)
}

// PutStars PUT /users/me/stars/:channelID
func PutStars(c echo.Context) error {
	userID := getRequestUserID(c)
	channelID := getRequestParamAsUUID(c, paramChannelID)

	if err := model.AddStar(userID, channelID); err != nil {
		switch err {
		case model.ErrNotFoundOrForbidden:
			return echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	go event.Emit(event.ChannelStared, &event.UserChannelEvent{UserID: userID, ChannelID: channelID})
	return c.NoContent(http.StatusNoContent)
}

// DeleteStars DELETE /users/me/stars/:channelID
func DeleteStars(c echo.Context) error {
	userID := getRequestUserID(c)
	channelID := getRequestParamAsUUID(c, paramChannelID)

	if _, err := validateChannelID(channelID, userID); err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	if err := model.RemoveStar(userID, channelID); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	go event.Emit(event.ChannelUnstared, &event.UserChannelEvent{UserID: userID, ChannelID: channelID})
	return c.NoContent(http.StatusNoContent)
}
