package v3

import (
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/router/extension/herror"
	"net/http"
)

// GetMyUnreadChannels GET /users/me/unread
func (h *Handlers) GetMyUnreadChannels(c echo.Context) error {
	userID := getRequestUserID(c)

	list, err := h.Repo.GetUserUnreadChannels(userID)
	if err != nil {
		return herror.InternalServerError(err)
	}

	return c.JSON(http.StatusOK, list)
}
