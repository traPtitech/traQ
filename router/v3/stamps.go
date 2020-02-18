package v3

import (
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"net/http"
)

// GetStamp GET /stamps/:stampID
func (h *Handlers) GetStamp(c echo.Context) error {
	return c.JSON(http.StatusOK, getParamStamp(c))
}

// DeleteStamp DELETE /stamps/:stampID
func (h *Handlers) DeleteStamp(c echo.Context) error {
	stampID := getParamAsUUID(c, consts.ParamStampID)

	if err := h.Repo.DeleteStamp(stampID); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}
