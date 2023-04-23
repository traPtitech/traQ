package middlewares

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
)

// RequestBodyLengthLimit リクエストボディのContentLengthで制限をかけるミドルウェア
func RequestBodyLengthLimit(kib int64) echo.MiddlewareFunc {
	limit := kib << 10
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if l := c.Request().Header.Get(echo.HeaderContentLength); len(l) == 0 {
				return echo.NewHTTPError(http.StatusLengthRequired) // ContentLengthを送ってこないリクエストを殺す
			}
			if c.Request().ContentLength > limit {
				return echo.NewHTTPError(http.StatusRequestEntityTooLarge, fmt.Sprintf("the request must be smaller than %dKiB", kib))
			}
			return next(c)
		}
	}
}
