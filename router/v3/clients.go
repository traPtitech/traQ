package v3

import (
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/router/extension/herror"
	"net/http"
)

// GetClient GET /clients/:clientID
func (h *Handlers) GetClient(c echo.Context) error {
	oc := getParamClient(c)

	if isTrue(c.QueryParam("detail")) {
		if oc.CreatorID != getRequestUserID(c) { // TODO 管理者権限
			return herror.Forbidden()
		}
		return c.JSON(http.StatusOK, formatOAuth2ClientDetail(oc))
	}

	return c.JSON(http.StatusOK, formatOAuth2Client(oc))
}

// DeleteClient DELETE /clients/:clientID
func (h *Handlers) DeleteClient(c echo.Context) error {
	oc := getParamClient(c)

	// delete client
	if err := h.Repo.DeleteClient(oc.ID); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}
