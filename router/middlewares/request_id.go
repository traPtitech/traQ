package middlewares

import (
	"github.com/labstack/echo/v4"

	"github.com/traPtitech/traQ/router/extension"
)

// RequestID リクエストIDを生成するミドルウェア
func RequestID() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Response().Header().Set(echo.HeaderXRequestID, extension.GetRequestID(c))
			return next(c)
		}
	}
}
