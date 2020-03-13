package v3

import (
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"net/http"
)

// DeleteClient DELETE /clients/:clientID
func (h *Handlers) DeleteClient(c echo.Context) error {
	clientID := c.Param(consts.ParamClientID)

	// delete client
	if err := h.Repo.DeleteClient(clientID); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}
