package router

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo"
	"github.com/labstack/echo-contrib/session"
	"github.com/traPtitech/traQ/model"
)

type MessageForResponse struct {
	MessageId       string
	UserId          string
	ParentChannelId string
	Content         string
	Datetime        string
	Pin             bool
	//StampList /*stampのオブジェクト*/
}

type requestMessage struct {
	Text string `json:"text"`
}

// GetMessageByIdHandler : /messages/{messageId}のGETメソッド
func GetMessageByIdHandler(c echo.Context) error {
	if _, err := getUserId(c); err != nil {
		return err
	}

	id := c.Param("messageId") // TODO: idの検証
	raw, err := model.GetMessage(id)
	if err != nil {
		errorMessageResponse(c, http.StatusNotFound, "Message is not found")
		return fmt.Errorf("model.Getmessage returned an error : %v", err)
	}
	res := formatMessage(raw)
	return c.JSON(http.StatusOK, res)
}

// GetMessagesByChannelIdHandler : /channels/{channelId}/messagesのGETメソッド
func GetMessagesByChannelIdHandler(c echo.Context) error {
	_, err := getUserId(c)
	if err != nil {
		return err
	}

	channelId := c.Param("channelId")

	messageList, err := model.GetMessagesFromChannel(channelId)
	if err != nil {
		errorMessageResponse(c, http.StatusNotFound, "Channel is not found")
		return fmt.Errorf("model.GetmessagesFromChannel returned an error : %v", err)
	}

	res := make(map[string]*MessageForResponse)

	for _, message := range messageList {
		res[message.Id] = formatMessage(message)
	}

	return c.JSON(http.StatusOK, values(res))
}

// PostMessageHandler : /channels/{cannelId}/messagesのPOSTメソッド
func PostMessageHandler(c echo.Context) error {
	userId, err := getUserId(c)
	if err != nil {
		return err
	}

	channelId := c.Param("ChannelId") //TODO: channelIdの検証

	post := new(requestMessage)
	if err := c.Bind(post); err != nil {
		errorMessageResponse(c, http.StatusBadRequest, "Invalid format")
		return fmt.Errorf("Invalid format: %v", err)
	}

	message := new(model.Messages)
	message.UserId = userId
	message.Text = post.Text
	message.ChannelId = channelId
	if err := message.Create(); err != nil {
		errorMessageResponse(c, http.StatusInternalServerError, "Failed to insert your message")
		return fmt.Errorf("Messages.Create() returned an error: %v", err)
	}
	return c.JSON(http.StatusCreated, formatMessage(message))
}

// PutMessageByIdHandler : /messages/{messageId}のPUTメソッド.メッセージの編集
func PutMessageByIdHandler(c echo.Context) error {
	userId, err := getUserId(c)
	if err != nil {
		return err
	}

	messageId := c.Param("messageId") //TODO: messageIdの検証

	req := new(requestMessage)
	if err := c.Bind(req); err != nil {
		errorMessageResponse(c, http.StatusBadRequest, "Invalid format")
		return fmt.Errorf("Request is invalid format: %v", err)
	}

	message, err := model.GetMessage(messageId)
	if err != nil {
		errorMessageResponse(c, http.StatusNotFound, "no message has the messageId: "+messageId)
		return fmt.Errorf("model.GetMessage() returned an error: %v", err)
	}

	message.Text = req.Text
	message.UpdaterId = userId
	if err := message.Update(); err != nil {
		errorMessageResponse(c, http.StatusInternalServerError, "Failed to update the message")
		return fmt.Errorf("message.Update() returned an error: %v", err)
	}

	return c.JSON(http.StatusOK, message)
}

// DeleteMessageByIdHandler : /message/{messageId}のDELETEメソッド.
func DeleteMessageByIdHandler(c echo.Context) error {
	if _, err := getUserId(c); err != nil {
		return err
	}
	// TODO:Userが権限を持っているかを確認

	messageId := c.Param("messageId")

	message, err := model.GetMessage(messageId)
	if err != nil {
		errorMessageResponse(c, http.StatusNotFound, "no message has the messageId: "+messageId)
		return fmt.Errorf("model.GetMessage() returned an error: %v", err)
	}

	message.IsDeleted = true
	if err := message.Update(); err != nil {
		errorMessageResponse(c, http.StatusInternalServerError, "Failed to update the message")
		return fmt.Errorf("message.Update() returned an error: %v", err)
	}
	return c.NoContent(http.StatusNoContent)
}

// 実質user認証みたいなことに使っている
func getUserId(c echo.Context) (string, error) {
	sess, err := session.Get("sessions", c)
	if err != nil {
		errorMessageResponse(c, http.StatusInternalServerError, "Failed to get a session")
		return "", fmt.Errorf("Failed to get a session: %v", err)
	}

	var userId string
	if sess.Values["userId"] != nil {
		userId = sess.Values["userId"].(string)
	} else {
		errorMessageResponse(c, http.StatusForbidden, "Your userId doesn't exist")
		return "", fmt.Errorf("This session doesn't have a userId")
	}
	return userId, nil
}

func values(m map[string]*MessageForResponse) []*MessageForResponse {
	val := []*MessageForResponse{}
	for _, v := range m {
		val = append(val, v)
	}
	return val
}

func formatMessage(raw *model.Messages) *MessageForResponse {
	res := MessageForResponse{
		MessageId:       raw.Id,
		UserId:          raw.UserId,
		ParentChannelId: raw.ChannelId,
		Pin:             false, //TODO:取得するようにする
		Content:         raw.Text,
		Datetime:        raw.CreatedAt,
	}
	//TODO: res.stampListの取得
	return &res
}
