package router

import (
	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/model"
	"net/http"
)

func GetMessageStamps(c echo.Context) error {
	messageID := c.Param("messageID")

	//TODO 見れないメッセージ(プライベートチャンネル)に対して404にする
	stamps, err := model.GetMessageStamps(messageID)
	if err != nil {
		c.Echo().Logger.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, stamps)
}

func PutMessageStamp(c echo.Context) error {
	userID := c.Get("user").(*model.User).ID
	messageID := c.Param("messageID")
	stampID := c.Param("stampID")

	//TODO 見れないメッセージ(プライベートチャンネル)に対して404にする
	err := model.AddStampToMessage(messageID, stampID, userID)
	if err != nil {
		//TODO エラーの種類で400,404,500に分岐
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	return c.NoContent(http.StatusNoContent)
}

func DeleteMessageStamp(c echo.Context) error {
	userID := c.Get("user").(*model.User).ID
	messageID := c.Param("messageID")
	stampID := c.Param("stampID")

	//TODO 見れないメッセージ(プライベートチャンネル)に対して404にする
	err := model.RemoveStampFromMessage(messageID, stampID, userID)
	if err != nil {
		//TODO エラーの種類で400,404,500に分岐
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	return c.NoContent(http.StatusNoContent)
}