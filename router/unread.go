package router

import (
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/event"
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

// DeleteUnread Method Handler of "DELETE /users/me/unread/{channelID}"
func DeleteUnread(c echo.Context) error {
	me := c.Get("user").(*model.User)

	channelID := c.Param("channelID")

	if err := model.DeleteUnreadsByChannelID(channelID, me.ID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete unread messages")
	}

	go event.Emit(event.MessageRead, &event.ReadMessageEvent{UserID: me.GetUID(), ChannelID: uuid.Must(uuid.FromString(channelID))})
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
