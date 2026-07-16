package middlewares

import (
	"github.com/labstack/echo/v5"
)

// ContentSecurityPolicy Content-Security-Policyヘッダーを追加するミドルウェア
func ContentSecurityPolicy() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			c.Response().Header().Set(echo.HeaderContentSecurityPolicy, "frame-ancestors 'none';")
			return next(c)
		}
	}
}
