package v3

import (
	"github.com/labstack/echo/v4"
	"net/http"
)

// GetVersion GET /version
func (h *Handlers) GetVersion(c echo.Context) error {
	return c.JSON(http.StatusOK, echo.Map{
		"version":  h.Version,
		"revision": h.Revision,
	})
}
