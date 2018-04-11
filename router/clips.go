package router

import (
	"fmt"
	"net/http"

	"github.com/traPtitech/traQ/notification"
	"github.com/traPtitech/traQ/notification/events"

	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/model"
)

// GetClips /users/me/clipsのGETメソッドハンドラ
func GetClips(c echo.Context) error {
	user := c.Get("user").(*model.User)

	clippedMessages, err := model.GetClippedMessages(user.ID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get clipped messages")
	}

	responseBody := make([]*MessageForResponse, 0)
	for _, message := range clippedMessages {
		responseBody = append(responseBody, formatMessage(message))
	}

	return c.JSON(http.StatusOK, responseBody)
}

// PostClips /users/me/clipsのPOSTメソッドハンドラ
func PostClips(c echo.Context) error {
	user := c.Get("user").(*model.User)

	requestBody := struct {
		MessageID string `json:"messageId"`
	}{}

	if err := c.Bind(&requestBody); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Failed to bind request body.")
	}

	// メッセージの存在確認
	if _, err := model.GetMessageByID(requestBody.MessageID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	clip := &model.Clip{
		UserID:    user.ID,
		MessageID: requestBody.MessageID,
	}

	if err := clip.Create(); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create clip: %s", err.Error()))
	}

	go notification.Send(events.MessageClipped, events.UserMessageEvent{UserID: user.ID, MessageID: requestBody.MessageID})
	return c.NoContent(http.StatusNoContent)
}

// DeleteClips /users/me/clipsのDELETEメソッドハンドラ
func DeleteClips(c echo.Context) error {
	user := c.Get("user").(*model.User)

	requestBody := struct {
		MessageID string `json:"messageId"`
	}{}

	if err := c.Bind(&requestBody); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Failed to bind request body.")
	}

	if _, err := validateMessageID(requestBody.MessageID, user.ID); err != nil {
		return err
	}

	clip := &model.Clip{
		UserID:    user.ID,
		MessageID: requestBody.MessageID,
	}

	if err := clip.Delete(); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to delete clip: %s", err.Error()))
	}

	go notification.Send(events.MessageUnclipped, events.UserMessageEvent{UserID: user.ID, MessageID: requestBody.MessageID})
	return c.NoContent(http.StatusNoContent)
}
