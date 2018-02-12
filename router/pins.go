package router

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo"
	"github.com/labstack/echo-contrib/session"
)

//ピン留めされているメッセージの取得
func GetPin(c echo.Context) error {
	sess, err := session.Get("sessions", c)
	if err != nil {
		return fmt.Errorf("Failed to get session: %v", err)
	}
	channelID := c.Param("channelId")

	pinnedMessage, err := model.GetPin(channelID)
	if err != nil {
		return fmt.Errorf("model.GetPin returned an error")
	}

	res := &MessageForResponse{
		MessageID:       pinnedMessage.MessageID,
		UserID:          pinnedMessage.UserID,
		ParentChannelID: pinnedMessage.ChannelID,
		Content:         "text message", //messageから呼び出す
		Datetime:        pinnedMessage.Datetime,
		Pin:             true,
	}
	c.JSON(http.StatusOK, res)

	return nil
}

func PutPin(c echo.Context) error {
	sess, err := session.Get("sessions", c)
	if err != nil {
		return fmt.Errorf("Failed to get session: %v", err)
	}
	var userId string
	if sees.Values["userId"] != nil {
		userID = sees.Values["userId"].(string)
	}

	channelID := c.Param("channelId")
	pin := &model.Pin{
		UserID:    userID,
		ChannelID: channelID,
		MessageID: requestBody.MessageID,
	}

	if err := pin.Create(); err != nil {
		c.Error(err)
		return err
	}
	pin, err := model.GetPin(channelID)
	if err != nil {
		c.Error(err)
		return err
	}

	res := &MessageForResponse{}

	return c.JSON(http.StatusCreated, pin)
}

func DeletePin(c ehco.Context) error {
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
		return fmt.Errorf("fail to get pin: %v", err)
	}

	if err := pin.DeletePin(); err != nil {
		return fmt.Errorf("fail to delete pin: %v", err)
	}

	return nil
}
