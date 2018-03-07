package router

import (
	"net/http"
	"time"

	"github.com/traPtitech/traQ/notification"
	"github.com/traPtitech/traQ/notification/events"

	"github.com/labstack/echo"
	"github.com/labstack/gommon/log"
	"github.com/traPtitech/traQ/model"
)

//MessageForResponse :クライアントに返す形のメッセージオブジェクト
type MessageForResponse struct {
	MessageID       string                `json:"messageId"`
	UserID          string                `json:"userId"`
	ParentChannelID string                `json:"parentChannelId"`
	Content         string                `json:"content"`
	Datetime        time.Time             `json:"datetime"`
	Pin             bool                  `json:"pin"`
	StampList       []*model.MessageStamp `json:"stampList"`
}

type requestMessage struct {
	Text string `json:"text"`
}

type requestCount struct {
	Limit  int `query:"limit"`
	Offset int `query:"offset"`
}

// GetMessageByID : /messages/{messageID}のGETメソッド
func GetMessageByID(c echo.Context) error {
	ID := c.Param("messageID") // TODO: idの検証
	raw, err := model.GetMessage(ID)
	if err != nil {
		c.Echo().Logger.Errorf("model.GetMessage returned an error : %v", err)
		return echo.NewHTTPError(http.StatusNotFound, "Message is not found")
	}
	res := formatMessage(raw)
	return c.JSON(http.StatusOK, res)
}

// GetMessagesByChannelID : /channels/{channelID}/messagesのGETメソッド
func GetMessagesByChannelID(c echo.Context) error {

	queryParam := &requestCount{}
	if err := c.Bind(queryParam); err != nil {
		c.Echo().Logger.Errorf("Invalid format: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid format")
	}

	channelID := c.Param("channelID")

	messageList, err := model.GetMessagesFromChannel(channelID, queryParam.Limit, queryParam.Offset)
	if err != nil {
		c.Echo().Logger.Errorf("model.GetMessagesFromChannel returned an error : %v", err)
		return echo.NewHTTPError(http.StatusNotFound, "Channel is not found")
	}

	res := make([]*MessageForResponse, 0)

	for _, message := range messageList {
		res = append(res, formatMessage(message))
	}

	return c.JSON(http.StatusOK, res)
}

// PostMessage : /channels/{channelID}/messagesのPOSTメソッド
func PostMessage(c echo.Context) error {
	userID := c.Get("user").(*model.User).ID
	channelID := c.Param("channelID") //TODO: channelIDの検証

	post := &requestMessage{}
	if err := c.Bind(post); err != nil {
		c.Echo().Logger.Errorf("Invalid format: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid format")
	}

	message := &model.Message{
		UserID:    userID,
		Text:      post.Text,
		ChannelID: channelID,
	}
	if err := message.Create(); err != nil {
		c.Echo().Logger.Errorf("Message.Create() returned an error: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to insert your message")
	}

	go notification.Send(events.MessageCreated, events.MessageEvent{Message: *message})
	return c.JSON(http.StatusCreated, formatMessage(message))
}

// PutMessageByID : /messages/{messageID}のPUTメソッド.メッセージの編集
func PutMessageByID(c echo.Context) error {
	userID := c.Get("user").(*model.User).ID
	messageID := c.Param("messageID") //TODO: messageIDの検証

	req := &requestMessage{}
	if err := c.Bind(req); err != nil {
		c.Echo().Logger.Errorf("Request is invalid format: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid format")
	}

	message, err := model.GetMessage(messageID)
	if err != nil {
		c.Echo().Logger.Errorf("model.GetMessage() returned an error: %v", err)
		return echo.NewHTTPError(http.StatusNotFound, "no message has the messageID: "+messageID)
	}

	message.Text = req.Text
	message.UpdaterID = userID
	if err := message.Update(); err != nil {
		c.Echo().Logger.Errorf("message.Update() returned an error: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update the message")
	}

	go notification.Send(events.MessageUpdated, events.MessageEvent{Message: *message})
	return c.JSON(http.StatusOK, message)
}

// DeleteMessageByID : /message/{messageID}のDELETEメソッド.
func DeleteMessageByID(c echo.Context) error {
	messageID := c.Param("messageID")

	message, err := model.GetMessage(messageID)
	if err != nil {
		c.Echo().Logger.Errorf("model.GetMessage() returned an error: %v", err)
		return echo.NewHTTPError(http.StatusNotFound, "no message has the messageID: "+messageID)
	}

	message.IsDeleted = true
	if err := message.Update(); err != nil {
		c.Echo().Logger.Errorf("message.Update() returned an error: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update the message")
	}

	if err := model.DeleteUnreadsByMessageID(messageID); err != nil {
		c.Echo().Logger.Errorf("model.DeleteUnreadsByMessageID returned an error: %v", err) //500エラーにはしない
	}

	go notification.Send(events.MessageDeleted, events.MessageEvent{Message: *message})
	return c.NoContent(http.StatusNoContent)
}

func valuesMessage(m map[string]*MessageForResponse) []*MessageForResponse {
	val := []*MessageForResponse{}
	for _, v := range m {
		val = append(val, v)
	}
	return val
}

func formatMessage(raw *model.Message) *MessageForResponse {
	stampList, err := model.GetMessageStamps(raw.ID)
	if err != nil {
		log.Error(err)
	}

	res := &MessageForResponse{
		MessageID:       raw.ID,
		UserID:          raw.UserID,
		ParentChannelID: raw.ChannelID,
		Pin:             false, //TODO:取得するようにする
		Content:         raw.Text,
		Datetime:        raw.CreatedAt.Truncate(time.Second).UTC(),
		StampList:       stampList,
	}
	return res
}
