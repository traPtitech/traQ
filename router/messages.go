package router

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo"
	"github.com/labstack/echo-contrib/session"
	"github.com/traPtitech/traQ/model"
)

type MessageForResponce struct {
	MessageId       string
	UserId          string
	ParentChannelId string
	Content         string
	Datetime        string
	Pin             bool
	//StampList /*stampのオブジェクト*/
}

type postMessage struct {
	Text string `json:"text"`
}

func GetMessageByIdHandler(c echo.Context) error {
	if _, err := getUserId(c); err != nil {
		return err
	}

	id := c.Param("messageId") // TODO: idの検証
	raw, err := model.GetMessage(id)
	if err != nil {
		errorMessageResponse(c, http.StatusNotFound, "Message is not found")
		return fmt.Errorf("model.Getmessage returns an error : %v", err)
	}
	res := formatMessgae(raw)
	return c.JSON(http.StatusOK, res)
}

func GetMessagesByChannelIdHandler(c echo.Context) error {
	return nil
}

// PostMessageHandler : /channels/{cannelId}/messagesのPOSTメソッド
func PostMessageHandler(c echo.Context) error {
	userId, err := getUserId(c)
	if err != nil {
		return err
	}

	channelId := c.Param("ChannelId")
	//TODO: channelIdの検証



	post := new(postMessage)
	if err := c.Bind(post); err != nil {
		errorMessageResponse(c, http.StatusBadRequest,"Invalid format")
		return fmt.Errorf("Invalid format: %v", err)
	}

	message := new(model.Messages)
	message.UserId = userId
	message.Text = post.Text
	message.ChannelId = channelId
	if err := message.Create(); err != nil {
		errorMessageResponse(c, http.StatusInternalServerError, "Failed to insert your message")
		return fmt.Errorf("Messages.Create() returns an error: %v", err)
	}
	return c.JSON(http.StatusCreated, formatMessgae(message))
}

func PutMessageByIdHandler(c echo.Context) error {
	return nil
}

func DeleteMessageByIdHandler(c echo.Context) error {

	return nil
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

func formatMessgae(raw *model.Messages) MessageForResponce {
	res := MessageForResponce{
		MessageId:       raw.Id,
		UserId:          raw.UserId,
		ParentChannelId: raw.ChannelId,
		Pin:             false, //TODO:取得するようにする
		Content:         raw.Text,
		Datetime:        raw.CreatedAt,
	}
	//TODO: res.pin,res.stampListの取得
	return res
}
