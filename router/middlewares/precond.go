package middlewares

import (
	"time"

	"github.com/labstack/echo/v4"

	"github.com/traPtitech/traQ/router/extension"
)

// CheckModTimePrecondition 事前条件検査ミドルウェア
func CheckModTimePrecondition(modTimeFunc func(c echo.Context) time.Time, preFunc ...echo.HandlerFunc) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if len(preFunc) > 0 {
				if err := preFunc[0](c); err != nil {
					return err
				}
			}
			modTime := modTimeFunc(c)
			extension.SetLastModified(c, modTime)
			if ok, _ := extension.CheckPreconditions(c, modTime); ok {
				return nil
			}
			return next(c)
		}
	}
}
