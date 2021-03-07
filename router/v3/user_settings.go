package v3

import (
	"net/http"

	"github.com/Azure/azure-amqp-common-go/v2/uuid"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/extension/herror"
)

// PutMyNotifyCitation PUT /user/me/settings/notify-citation
func (h *Handlers) PutMyNotifyCitation(c echo.Context) error {
	id := getRequestUserID(c)
	us := getParamUserSettings(c)
	err := h.Repo.UpdateNotifyCitation(id, us.NotifyCitation)

	if err != nil {
		switch {
		case err == uuid.Nil:
			return herror.BadRequest("Invalid url")
		case err == repository.ErrNotFound:
			return herror.NotFound("User not found")
		default:
			return herror.InternalServerError(err)
		}
	}

	return c.NoContent(http.StatusNoContent)

}

// GetMySettings GET /user/me/settings
func (h *Handlers) GetMySettings(c echo.Context) error {
	id := getRequestUserID(c)
	us, err := h.Repo.GetUserSettings(id)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, us)
}

// GetMyNotifyCitation GET /user/me/settings/notify-citation
func (h *Handlers) GetMyNotifyCitation(c echo.Context) error {
	id := getRequestUserID(c)
	nc, err := h.Repo.GetNotifyCitation(id)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, nc)
}
