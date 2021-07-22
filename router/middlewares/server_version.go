package middlewares

import (
	"github.com/labstack/echo/v4"

	"github.com/traPtitech/traQ/router/consts"
)

// ServerVersion X-TRAQ-VERSIONレスポンスヘッダーを追加するミドルウェア
func ServerVersion(version string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Response().Header().Set(consts.HeaderVersion, version)
			return next(c)
		}
	}
}
