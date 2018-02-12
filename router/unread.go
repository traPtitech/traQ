package router

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/model"
)

// GetUnread: Method Handler of "GET /users/me/unread"
func GetUnread(c echo.Context) error {
	me := c.Get("user").(*model.User)

	responseBody, err := getUnreadResponse(me.ID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get unread.")
	}

	return c.JSON(http.StatusOK, responseBody)
}

// DeleteUnread: Method Handler of "DELETE /users/me/unread"
func DeleteUnread(c echo.Context) error {
	me := c.Get("user").(*model.User)

	requestBody := struct {
		MessageIDs []string
	}{}

	if err := c.Bind(&requestBody); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Failed to bind request body.")
	}

	for _, messageID := range requestBody.MessageIDs {
		unread := &model.Unread{
			UserID:    me.ID,
			MessageID: messageID,
		}

		if err := unread.Delete(); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to delete unread: %v", err))
		}
	}

	return c.NoContent(http.StatusNoContent)
}

func getUnreadResponse(userID string) ([]*MessageForResponse, error) {
	unreads, err := model.GetUnreadsByUserID(userID)
	if err != nil {
		return nil, err
	}

	responseBody := make([]*MessageForResponse, 0)
	for _, unread := range unreads {
		message, err := model.GetMessage(unread.MessageID)
		if err != nil {
			return nil, err
		}
		responseBody = append(responseBody, formatMessage(message))
	}
	return responseBody, nil
}
