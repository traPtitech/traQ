package router

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/model"
)

// GetClips /users/me/clipsのGETメソッドハンドラ
func GetClips(c echo.Context) error {
	user := c.Get("user").(*model.User)

	clipedMessages, err := model.GetClippedMessages(user.ID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get cliped messages")
	}

	responseBody := make([]*MessageForResponse, 0)
	for _, message := range clipedMessages {
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
	if _, err := model.GetMessage(requestBody.MessageID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	clip := &model.Clip{
		UserID:    user.ID,
		MessageID: requestBody.MessageID,
	}

	if err := clip.Create(); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create clip: %s", err.Error()))
	}

	clipedMessages, err := model.GetClippedMessages(user.ID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get cliped messages")
	}

	responseBody := make([]*MessageForResponse, 0)
	for _, message := range clipedMessages {
		responseBody = append(responseBody, formatMessage(message))
	}

	return c.JSON(http.StatusCreated, responseBody)
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

	clip := &model.Clip{
		UserID:    user.ID,
		MessageID: requestBody.MessageID,
	}

	if err := clip.Delete(); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to delete clip: %s", err.Error()))
	}

	clipedMessages, err := model.GetClippedMessages(user.ID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get cliped messages")
	}

	responseBody := make([]*MessageForResponse, 0)
	for _, message := range clipedMessages {
		responseBody = append(responseBody, formatMessage(message))
	}

	return c.JSON(http.StatusOK, responseBody)
}
