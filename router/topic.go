package router

import (
	"net/http"

	"github.com/traPtitech/traQ/notification"
	"github.com/traPtitech/traQ/notification/events"

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
		return err
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
		return err
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

	go notification.Send(events.ChannelUpdated, events.ChannelEvent{ID: channelID})
	return c.JSON(http.StatusOK, topic)
}
