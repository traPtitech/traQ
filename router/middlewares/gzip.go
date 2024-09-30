package middlewares

import (
	"compress/gzip"
	"net/http"

	"github.com/NYTimes/gziphandler"
	"github.com/labstack/echo/v4"
)

// Gzip Gzipミドルウェア
func Gzip() echo.MiddlewareFunc {
	gzh, _ := gziphandler.GzipHandlerWithOpts(
		gziphandler.ContentTypes([]string{
			"application/javascript",
			"application/json",
			"image/svg+xml",
			"text/css",
			"text/html",
			"text/plain",
			"text/xml",
		}),
		gziphandler.CompressionLevel(gzip.BestSpeed),
	)
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			gzh(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				c.SetRequest(r)
				c.Response().Writer = w
				if err := next(c); err != nil {
					c.Error(err)
				}
			})).ServeHTTP(c.Response().Writer, c.Request())
			return
		}
	}
}
