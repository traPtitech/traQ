package v3

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// PutUserNotifyCitation PUT /user/:userID/settings/notify-citation
func (h *Handlers) PutUserNotifyCitation(c echo.Context) error {
	id := getRequestUserID(c)
	us := getParamUserSettings(c)
	err := h.Repo.UpdateNotifyCitation(id, us.NotifyCitation)

	if err != nil {
		return err
	}

	return c.NoContent(http.StatusNoContent)

}

// GetUserSettings GET /user/:userID/settings
func (h *Handlers) GetUserSettings(c echo.Context) error {
	id := getRequestUserID(c)
	us, err := h.Repo.GetUserSettings(id)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, us)
}

// GetUserNotifyCitation GET /user/:userID/settings/notify-citation
func (h *Handlers) GetUserNotifyCitation(c echo.Context) error {
	id := getRequestUserID(c)
	nc, err := h.Repo.GetNotifyCitation(id)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, nc)
}
