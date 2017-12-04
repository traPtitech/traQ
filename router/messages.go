package router

import (
	"github.com/labstack/echo"
)

type MessageForResponce struct {
	MessageId string
	User string
	ParentChannelId string
	Content string
	Datetime string
	//StampList /*stampのオブジェクト*/
}

type postMessage struct {
	text string `json:"text"`
}


func GetMessageByIdHandler(c echo.Context) error {
	return nil
}

func GetMessagesByChannelIdHandler(c echo.Context) error {
	return nil
}

func PostMessageHandler(c echo.Context) error {
	return nil
}

func PutMessageByIdHandler(c echo.Context) error {
	return nil
}

func DeleteMessageByIdHandler(c echo.Context) error {
	return nil
}