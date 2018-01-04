package router

import (
	"fmt"
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
	channelID := c.Param("channelID")
	channel := &model.Channel{
		ID: channelID,
	}
	has, err := channel.Exists(userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "An error occurred while check channelID.")
	}

	if !has {
		return echo.NewHTTPError(http.StatusNotFound, "Channel not found.")
	}

	topic := TopicForResponse{
		ChannelID: channel.ID,
		Name:      channel.Name,
		Text:      channel.Topic,
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
	channel := &model.Channel{
		ID: channelID,
	}
	has, err := channel.Exists(userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "An error occurred while check channelID.")
	}

	if !has {
		return echo.NewHTTPError(http.StatusNotFound, "Channel not found.")
	}

	channel.Topic = requestBody.Text
	channel.UpdaterID = userID

	fmt.Println(userID)
	if err := channel.Update(); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "An error occuerred when channel model update.")
	}

	topic := TopicForResponse{
		ChannelID: channel.ID,
		Name:      channel.Name,
		Text:      channel.Topic,
	}
	return c.JSON(http.StatusOK, topic)
}
