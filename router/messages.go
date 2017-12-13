package router

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo"
	"github.com/labstack/echo-contrib/session"
	"github.com/traPtitech/traQ/model"
)

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

// GetMessageByIDHandler : /messages/{messageID}のGETメソッド
func GetMessageByIDHandler(c echo.Context) error {
	if _, err := getUserID(c); err != nil {
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

// GetMessagesByChannelIDHandler : /channels/{channelID}/messagesのGETメソッド
func GetMessagesByChannelIDHandler(c echo.Context) error {
	_, err := getUserID(c)
	if err != nil {
		return err
	}

	channelID := c.Param("channelId")

	messageList, err := model.GetMessagesFromChannel(channelID)
	if err != nil {
		errorMessageResponse(c, http.StatusNotFound, "Channel is not found")
		return fmt.Errorf("model.GetmessagesFromChannel returned an error : %v", err)
	}

	res := make(map[string]*MessageForResponse)

	for _, message := range messageList {
		res[message.ID] = formatMessage(message)
	}

	return c.JSON(http.StatusOK, values(res))
}

// PostMessageHandler : /channels/{cannelID}/messagesのPOSTメソッド
func PostMessageHandler(c echo.Context) error {
	userID, err := getUserID(c)
	if err != nil {
		return err
	}

	channelID := c.Param("ChannelId") //TODO: channelIDの検証

	post := new(requestMessage)
	if err := c.Bind(post); err != nil {
		errorMessageResponse(c, http.StatusBadRequest, "Invalid format")
		return fmt.Errorf("Invalid format: %v", err)
	}

	message := new(model.Messages)
	message.UserID = userID
	message.Text = post.Text
	message.ChannelID = channelID
	if err := message.Create(); err != nil {
		errorMessageResponse(c, http.StatusInternalServerError, "Failed to insert your message")
		return fmt.Errorf("Messages.Create() returned an error: %v", err)
	}
	return c.JSON(http.StatusCreated, formatMessage(message))
}

// PutMessageByIDHandler : /messages/{messageID}のPUTメソッド.メッセージの編集
func PutMessageByIDHandler(c echo.Context) error {
	userID, err := getUserID(c)
	if err != nil {
		return err
	}

	messageID := c.Param("messageId") //TODO: messageIDの検証

	req := new(requestMessage)
	if err := c.Bind(req); err != nil {
		errorMessageResponse(c, http.StatusBadRequest, "Invalid format")
		return fmt.Errorf("Request is invalid format: %v", err)
	}

	message, err := model.GetMessage(messageID)
	if err != nil {
		errorMessageResponse(c, http.StatusNotFound, "no message has the messageID: "+messageID)
		return fmt.Errorf("model.GetMessage() returned an error: %v", err)
	}

	message.Text = req.Text
	message.UpdaterID = userID
	if err := message.Update(); err != nil {
		errorMessageResponse(c, http.StatusInternalServerError, "Failed to update the message")
		return fmt.Errorf("message.Update() returned an error: %v", err)
	}

	return c.JSON(http.StatusOK, message)
}

// DeleteMessageByIDHandler : /message/{messageID}のDELETEメソッド.
func DeleteMessageByIDHandler(c echo.Context) error {
	if _, err := getUserID(c); err != nil {
		return err
	}
	// TODO:Userが権限を持っているかを確認

	messageID := c.Param("messageId")

	message, err := model.GetMessage(messageID)
	if err != nil {
		errorMessageResponse(c, http.StatusNotFound, "no message has the messageID: "+messageID)
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
func getUserID(c echo.Context) (string, error) {
	sess, err := session.Get("sessions", c)
	if err != nil {
		errorMessageResponse(c, http.StatusInternalServerError, "Failed to get a session")
		return "", fmt.Errorf("Failed to get a session: %v", err)
	}

	var userID string
	if sess.Values["userId"] != nil {
		userID = sess.Values["userId"].(string)
	} else {
		errorMessageResponse(c, http.StatusForbidden, "Your userID doesn't exist")
		return "", fmt.Errorf("This session doesn't have a userId")
	}
	return userID, nil
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
