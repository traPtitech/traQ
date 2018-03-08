package router

import (
	"fmt"
	"github.com/traPtitech/traQ/notification"
	"github.com/traPtitech/traQ/notification/events"
	"net/http"

	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/model"
)

// GetUnread Method Handler of "GET /users/me/unread"
func GetUnread(c echo.Context) error {
	me := c.Get("user").(*model.User)

	responseBody, err := getUnreadResponse(me.ID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get unread.")
	}

	return c.JSON(http.StatusOK, responseBody)
}

// DeleteUnread Method Handler of "DELETE /users/me/unread"
func DeleteUnread(c echo.Context) error {
	me := c.Get("user").(*model.User)

	var requestBody []string

	if err := c.Bind(&requestBody); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Failed to bind request body.")
	}

	for _, messageID := range requestBody {
		unread := &model.Unread{
			UserID:    me.ID,
			MessageID: messageID,
		}

		if err := unread.Delete(); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to delete unread: %v", err))
		}
	}

	go notification.Send(events.MessageRead, events.ReadMessagesEvent{UserID: me.ID, MessageIDs: requestBody})
	return c.NoContent(http.StatusNoContent)
}

func getUnreadResponse(userID string) ([]*MessageForResponse, error) {
	unreads, err := model.GetUnreadsByUserID(userID)
	if err != nil {
		return nil, err
	}

	responseBody := make([]*MessageForResponse, 0)
	for _, unread := range unreads {
		message, err := model.GetMessageByID(unread.MessageID)
		if err != nil {
			return nil, err
		}
		res := formatMessage(message)
		responseBody = append(responseBody, res)
	}
	return responseBody, nil
}
