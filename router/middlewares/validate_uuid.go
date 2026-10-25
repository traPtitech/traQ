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
			if _, err := parseUUIDParam(c.Param(param)); err != nil {
				return err
			}
			return next(c)
		}
	}
}

func parseUUIDParam(value string) (uuid.UUID, error) {
	id, err := uuid.FromString(value)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, herror.NotFound()
	}
	return id, nil
}
