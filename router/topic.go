package router

import (
	"github.com/traPtitech/traQ/event"
	"net/http"

	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/model"
)

// GetTopic GET /channels/:channelID/topic
func GetTopic(c echo.Context) error {
	userID := getRequestUserID(c)
	channelID := getRequestParamAsUUID(c, paramChannelID)

	ch, err := validateChannelID(channelID, userID)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound, "this channel is not found")
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	return c.JSON(http.StatusOK, map[string]string{
		"text": ch.Topic,
	})
}

// PutTopic PUT /channels/:channelID/topic
func PutTopic(c echo.Context) error {
	userID := getRequestUserID(c)
	channelID := getRequestParamAsUUID(c, paramChannelID)

	ch, err := validateChannelID(channelID, userID)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound, "this channel is not found")
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	req := struct {
		Text string `json:"text"`
	}{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	if err := model.UpdateChannelTopic(channelID, req.Text, userID); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	if ch.IsPublic {
		go event.Emit(event.ChannelUpdated, &event.ChannelEvent{ID: ch.ID})
	} else {
		go event.Emit(event.ChannelUpdated, &event.PrivateChannelEvent{ChannelID: channelID})
	}

	return c.NoContent(http.StatusNoContent)
}
