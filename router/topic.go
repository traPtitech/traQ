package router

import (
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/event"
	"net/http"

	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/model"
)

// TopicForResponse レスポンス用構造体
type TopicForResponse struct {
	ChannelID string `json:"channelId"`
	Name      string `json:"name"`
	Text      string `json:"text"`
}

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

	topic := TopicForResponse{
		ChannelID: ch.ID,
		Name:      ch.Name,
		Text:      ch.Topic,
	}
	return c.JSON(http.StatusOK, topic)
}

// PutTopic PUT /channels/{channelID}/topic
func PutTopic(c echo.Context) error {
	userID := c.Get("user").(*model.User).ID

	type putTopic struct {
		Text string `json:"text"`
	}
	requestBody := putTopic{}
	c.Bind(&requestBody)

	channelID := c.Param("channelID")
	ch, err := validateChannelID(channelID, userID)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound, "this channel is not found")
		default:
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to find the specified channel")
		}
	}

	ch.Topic = requestBody.Text
	ch.UpdaterID = userID

	if err := ch.Update(); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "An error occurred when channel model update.")
	}

	topic := TopicForResponse{
		ChannelID: ch.ID,
		Name:      ch.Name,
		Text:      ch.Topic,
	}

	if ch.IsPublic {
		go event.Emit(event.ChannelUpdated, event.ChannelEvent{ID: channelID})
	} else {
		users, err := model.GetPrivateChannelMembers(channelID)
		if err != nil {
			c.Logger().Error(err)
		}
		ids := make([]uuid.UUID, len(users))
		for i, v := range users {
			ids[i] = uuid.Must(uuid.FromString(v))
		}
		go event.Emit(event.ChannelUpdated, event.PrivateChannelEvent{UserIDs: ids, ChannelID: uuid.Must(uuid.FromString(channelID))})
	}

	return c.JSON(http.StatusOK, topic)
}
