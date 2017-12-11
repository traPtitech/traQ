package router

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo"
	"github.com/traptitech/traQ/model"

)

type PinnedMessageForResponse struct {
	MessageId       string
	UserId          string
	ParentChannelId string
	Content         string
	Datetime        string
	Pin             bool
}
type ReqPutPin struct {
	messageId string
}

//ピン留めされているメッセージの取得
func GetPinHandler(c echo.Context) error {
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
}

func DeletePinHandler(c ehco.Context) {

}
