package middlewares

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var requestCounter = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: "traq",
	Name:      "http_requests_total",
}, []string{"code", "method"})

// RequestCounter prometheus metrics用リクエストカウンター
func RequestCounter() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) (err error) {
			err = next(c)
			status := http.StatusOK
			if res, uerr := echo.UnwrapResponse(c.Response()); uerr == nil {
				status = res.Status
			}
			requestCounter.WithLabelValues(strconv.Itoa(status), c.Request().Method).Inc()
			return err
		}
	}
}
