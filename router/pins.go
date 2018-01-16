package router

import (
	"fmt"
	"net/http"
	"github.com/labstack/echo"
	"github.com/traptitech/traQ/model"

)

type Pin struct {
	MessageId       string
	UserId          string
	ParentChannelId string
	Content         string
	Datetime        string
	Pin             bool
}

//ピン留めされているメッセージの取得
func GetPinHandler(c echo.Context) error {
	sess, err := session.Get("sessions", c)
	if err != nil {
		return fmt.Errorf("Failed to get session: %v", err)
	}
	var userId string
	if sees.Values["userId"] != nil {
		userId = sees.Values["userId"].(string)
	}


	channelId := c.Param("channelId")

	pinnedMessage, err := model.GetPinedMege(channelId)
	if err != nil {
		return fmt.Errorf("model.GetPin returned an error")
	}

	res := {
		MessageId : pinnedMessage.MessageId,
		UserId : pinnedMessage.UserId,
		ParentChannelId : pinnedMessage.ChannelId,
		Content : "text message",
		Datetime : "date time",
		Pin : true,
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
	var requestBody Pin
	c.Bind(&requestBody)


	channelId := c.Param("channelId")
	pin := &model.Pins {
		UserId : userId,
		ChannelId : channelId,
		MessageId : requestBody.MessageId,

	}

	if err := newChannenl.Create(); err != nil {
		c.Error(err)
		return err
	}
	ch, err := model.GetPin(channelId)
	if err != nil {
		c.Error(err)
		return err
	}

	return c.JSON(http.StatusCreated, ch)
}

func DeletePinHandler(c ehco.Context) error{
	userID , err := getUserID(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("An error occurred while getUserIS: %v", err))
	}
	type messageID struct {
		MessageID string `json:"messageId"`
	}
	var requestBody messageID
	c.Bind(&requestBody)

	channelID := c.Param("channelID")

	return error;

}
