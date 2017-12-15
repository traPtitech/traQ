package router

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/model"
)

//MessageForResponse :クライアントに返す形のメッセージオブジェクト
type MessageForResponse struct {
	MessageID       string
	UserID          string
	ParentChannelID string
	Content         string
	Datetime        string
	Pin             bool
	//StampList /*stampのオブジェクト*/
}

type requestMessage struct {
	Text string `json:"text"`
}

// GetMessageByID : /messages/{messageID}のGETメソッド
func GetMessageByID(c echo.Context) error {
	if _, err := getUserID(c); err != nil {
		return echo.NewHTTPError(http.StatusForbidden, "your id is not found")
	}

	id := c.Param("messageId") // TODO: idの検証
	raw, err := model.GetMessage(id)
	if err != nil {
		fmt.Errorf("model.Getmessage returned an error : %v", err)
		return echo.NewHTTPError(http.StatusNotFound, "Message is not found")
	}
	res := formatMessage(raw)
	return c.JSON(http.StatusOK, res)
}

// GetMessagesByChannelID : /channels/{channelID}/messagesのGETメソッド
func GetMessagesByChannelID(c echo.Context) error {
	_, err := getUserID(c)
	if err != nil {
		return err
	}

	channelID := c.Param("channelId")

	messageList, err := model.GetMessagesFromChannel(channelID)
	if err != nil {
		fmt.Errorf("model.GetmessagesFromChannel returned an error : %v", err)
		return echo.NewHTTPError(http.StatusNotFound, "Channel is not found")
	}

	res := make(map[string]*MessageForResponse)

	for _, message := range messageList {
		res[message.ID] = formatMessage(message)
	}

	return c.JSON(http.StatusOK, values(res))
}

// PostMessage : /channels/{cannelID}/messagesのPOSTメソッド
func PostMessage(c echo.Context) error {
	userID, err := getUserID(c)
	if err != nil {
		return err
	}

	channelID := c.Param("ChannelId") //TODO: channelIDの検証

	post := &requestMessage{}
	if err := c.Bind(post); err != nil {
		fmt.Errorf("Invalid format: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid format")
	}

	message := &model.Message{
		UserID:    userID,
		Text:      post.Text,
		ChannelID: channelID,
	}
	if err := message.Create(); err != nil {
		fmt.Errorf("Message.Create() returned an error: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to insert your message")
	}
	return c.JSON(http.StatusCreated, formatMessage(message))
}

// PutMessageByID : /messages/{messageID}のPUTメソッド.メッセージの編集
func PutMessageByID(c echo.Context) error {
	userID, err := getUserID(c)
	if err != nil {
		return err
	}

	messageID := c.Param("messageId") //TODO: messageIDの検証

	req := &requestMessage{}
	if err := c.Bind(req); err != nil {
		fmt.Errorf("Request is invalid format: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid format")
	}

	message, err := model.GetMessage(messageID)
	if err != nil {
		fmt.Errorf("model.GetMessage() returned an error: %v", err)
		return echo.NewHTTPError(http.StatusNotFound, "no message has the messageID: "+messageID)
	}

	message.Text = req.Text
	message.UpdaterID = userID
	if err := message.Update(); err != nil {
		fmt.Errorf("message.Update() returned an error: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update the message")
	}

	return c.JSON(http.StatusOK, message)
}

// DeleteMessageByID : /message/{messageID}のDELETEメソッド.
func DeleteMessageByID(c echo.Context) error {
	if _, err := getUserID(c); err != nil {
		return err
	}
	// TODO:Userが権限を持っているかを確認

	messageID := c.Param("messageId")

	message, err := model.GetMessage(messageID)
	if err != nil {
		fmt.Errorf("model.GetMessage() returned an error: %v", err)
		return echo.NewHTTPError(http.StatusNotFound, "no message has the messageID: "+messageID)
	}

	message.IsDeleted = true
	if err := message.Update(); err != nil {
		fmt.Errorf("message.Update() returned an error: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update the message")
	}
	return c.NoContent(http.StatusNoContent)
}

func values(m map[string]*MessageForResponse) []*MessageForResponse {
	val := []*MessageForResponse{}
	for _, v := range m {
		val = append(val, v)
	}
	return val
}

func formatMessage(raw *model.Message) *MessageForResponse {
	res := MessageForResponse{
		MessageID:       raw.ID,
		UserID:          raw.UserID,
		ParentChannelID: raw.ChannelID,
		Pin:             false, //TODO:取得するようにする
		Content:         raw.Text,
		Datetime:        raw.CreatedAt,
	}
	//TODO: res.stampListの取得
	return &res
}
