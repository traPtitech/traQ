package middlewares

import (
	"github.com/blendle/zapdriver"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/router/extension"
	"go.uber.org/zap"
	"strconv"
	"strings"
	"time"
)

// AccessLogging アクセスログミドルウェア
func AccessLogging(logger *zap.Logger, dev bool) echo.MiddlewareFunc {
	if dev {
		return func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				start := time.Now()
				if err := next(c); err != nil {
					c.Error(err)
				}
				stop := time.Now()

				req := c.Request()
				res := c.Response()
				logger.Sugar().Infof("%3d | %s | %s %s %d", res.Status, stop.Sub(start), req.Method, req.URL, res.Size)
				return nil
			}
		}
	}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if strings.HasPrefix(c.Path(), "/api/1.0/heartbeat") {
				return next(c)
			}

			start := time.Now()
			if err := next(c); err != nil {
				c.Error(err)
			}
			stop := time.Now()

			req := c.Request()
			res := c.Response()
			logger.Info("", zap.String("logging.googleapis.com/trace", extension.GetTraceID(c)), zapdriver.HTTP(&zapdriver.HTTPPayload{
				RequestMethod: req.Method,
				Status:        res.Status,
				UserAgent:     req.UserAgent(),
				RemoteIP:      c.RealIP(),
				Referer:       req.Referer(),
				Protocol:      req.Proto,
				RequestURL:    req.URL.String(),
				RequestSize:   req.Header.Get(echo.HeaderContentLength),
				ResponseSize:  strconv.FormatInt(res.Size, 10),
				Latency:       strconv.FormatFloat(stop.Sub(start).Seconds(), 'f', 9, 64) + "s",
			}))
			return nil
		}
	}
}
