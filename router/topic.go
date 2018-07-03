package router

import (
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/event"
	"net/http"

	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/model"
)

// GetTopic GET /channels/{channelID}/topic
func GetTopic(c echo.Context) error {
	userID := c.Get("user").(*model.User).ID
	ch, err := validateChannelID(c.Param("channelID"), userID)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound, "this channel is not found")
		default:
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to find the specified channel")
		}
	}

	return c.JSON(http.StatusOK, map[string]string{
		"text": ch.Topic,
	})
}

// PutTopic PUT /channels/{channelID}/topic
func PutTopic(c echo.Context) error {
	user := c.Get("user").(*model.User)

	req := struct {
		Text string `json:"text"`
	}{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	channelID := c.Param("channelID")
	ch, err := validateChannelID(channelID, user.ID)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound, "this channel is not found")
		default:
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to find the specified channel")
		}
	}

	if err := model.UpdateChannelTopic(ch.GetCID(), req.Text, user.GetUID()); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, "An error occurred when channel model update.")
	}

	if ch.IsPublic {
		go event.Emit(event.ChannelUpdated, &event.ChannelEvent{ID: ch.ID})
	} else {
		users, err := model.GetPrivateChannelMembers(ch.ID)
		if err != nil {
			c.Logger().Error(err)
		}
		ids := make([]uuid.UUID, len(users))
		for i, v := range users {
			ids[i] = uuid.Must(uuid.FromString(v))
		}
		go event.Emit(event.ChannelUpdated, &event.PrivateChannelEvent{UserIDs: ids, ChannelID: ch.GetCID()})
	}

	return c.NoContent(http.StatusNoContent)
}
