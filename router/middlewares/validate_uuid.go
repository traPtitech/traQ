package middlewares

import (
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v5"

	"github.com/traPtitech/traQ/router/extension/herror"
)

// ValidateUUIDParam パスパラメータがnil以外のUUIDであることを確認します。
func ValidateUUIDParam(param string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			id, err := uuid.FromString(c.Param(param))
			if err != nil || id == uuid.Nil {
				return herror.NotFound()
			}
			return next(c)
		}
	}
}
