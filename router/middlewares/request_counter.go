package middlewares

import (
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"strconv"
)

var requestCounter = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: "traq",
	Name:      "http_requests_total",
}, []string{"code", "method"})

// RequestCounter prometheus metrics用リクエストカウンター
func RequestCounter() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			err = next(c)
			requestCounter.WithLabelValues(strconv.Itoa(c.Response().Status), c.Request().Method).Inc()
			return err
		}
	}
}
