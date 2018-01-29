package router

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo"
	"github.com/labstack/echo-contrib/session"
)

type PinForResponse struct {
	MessageID       string `json:"messageId"`
	UserID          string `json:"userId"`
	ParentChannelID string `json:"parentChannelId"`
	Content         string `json:"content"`
	Datetime        string `json:"datetime"`
	Pin             bool   `json:"pin"`
}

//ピン留めされているメッセージの取得
func GetPinHandler(c echo.Context) error {
	sess, err := session.Get("sessions", c)
	if err != nil {
		return fmt.Errorf("Failed to get session: %v", err)
	}
	channelId := c.Param("channelId")

	pinnedMessage, err := model.GetPin(channelId)
	if err != nil {
		return fmt.Errorf("model.GetPin returned an error")
	}

	res := &MessageForResponse{
		MessageId:       pinnedMessage.MessageId,
		UserId:          pinnedMessage.UserId,
		ParentChannelId: pinnedMessage.ChannelId,
		Content:         "text message", //messageから呼び出す
		Datetime:        pinnedMessage.Datetime,
		Pin:             true,
	}
	c.JSON(http.StatusOK, res)

	return nil
}

func PutPinHandler(c echo.Context) error {
	sess, err := session.Get("sessions", c)
	if err != nil {
		return fmt.Errorf("Failed to get session: %v", err)
	}
	var userId string
	if sees.Values["userId"] != nil {
		userId = sees.Values["userId"].(string)
	}

	channelId := c.Param("channelId")
	pin := &model.Pins{
		UserId:    userId,
		ChannelId: channelId,
		MessageId: requestBody.MessageId,
	}

	if err := pin.Create(); err != nil {
		c.Error(err)
		return err
	}
	pin, err := model.GetPin(channelId)
	if err != nil {
		c.Error(err)
		return err
	}

	res := &MessageForResponse{}

	return c.JSON(http.StatusCreated, pin)
}

func DeletePinHandler(c ehco.Context) error {
	userID, err := getUserID(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("An error occurred while getUserIS: %v", err))
	}
	type messageID struct {
		MessageID string `json:"messageId"`
	}

	var requestBody messageID
	c.Bind(&requestBody)

	channelID := c.Param("channelID")

	pin, err := model.GetPin(channelID, messageID)

	if err != nil {
		return c.Error(err)
	}
	return error

}
