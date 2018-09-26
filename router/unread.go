package router

import (
	"github.com/traPtitech/traQ/event"
	"net/http"

	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/model"
)

// GetUnread GET /users/me/unread
func GetUnread(c echo.Context) error {
	userID := getRequestUserID(c)

	unreads, err := model.GetUnreadMessagesByUserID(userID)
	if err != nil {
		c.Logger().Error()
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	responseBody := make([]*MessageForResponse, len(unreads))
	for i, v := range unreads {
		responseBody[i] = formatMessage(v)
	}

	return c.JSON(http.StatusOK, responseBody)
}

// DeleteUnread DELETE /users/me/unread/:channelID
func DeleteUnread(c echo.Context) error {
	userID := getRequestUserID(c)
	channelID := getRequestParamAsUUID(c, paramChannelID)

	if err := model.DeleteUnreadsByChannelID(channelID, userID); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	go event.Emit(event.MessageRead, &event.ReadMessageEvent{UserID: userID, ChannelID: channelID})
	return c.NoContent(http.StatusNoContent)
}
