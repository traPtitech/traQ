package middlewares

import (
	"strconv"
	"time"

	"github.com/blendle/zapdriver"
	"github.com/labstack/echo/v5"
	"go.uber.org/zap"

	"github.com/traPtitech/traQ/router/extension"
)

// responseStatusSize レスポンスのステータスコードとサイズを返す。
// echo.UnwrapResponseが*echo.Responseを取り出せない場合は(0, 0)を返す。
func responseStatusSize(c *echo.Context) (status int, size int64) {
	if res, err := echo.UnwrapResponse(c.Response()); err == nil {
		return res.Status, res.Size
	}
	return 0, 0
}

// AccessLogging アクセスログミドルウェア
func AccessLogging(logger *zap.Logger, dev bool) echo.MiddlewareFunc {
	if dev {
		return func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c *echo.Context) error {
				start := time.Now()
				if err := next(c); err != nil {
					c.Echo().HTTPErrorHandler(c, err)
				}
				stop := time.Now()

				req := c.Request()
				status, size := responseStatusSize(c)
				logger.Sugar().Infof("%3d | %s | %s %s %d", status, stop.Sub(start), req.Method, req.URL, size)
				return nil
			}
		}
	}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			start := time.Now()
			if err := next(c); err != nil {
				c.Echo().HTTPErrorHandler(c, err)
			}
			stop := time.Now()

			req := c.Request()
			status, size := responseStatusSize(c)
			logger.Info("",
				zap.String("requestId", extension.GetRequestID(c)),
				zapdriver.HTTP(&zapdriver.HTTPPayload{
					RequestMethod: req.Method,
					Status:        status,
					UserAgent:     req.UserAgent(),
					RemoteIP:      c.RealIP(),
					Referer:       req.Referer(),
					Protocol:      req.Proto,
					RequestURL:    req.URL.String(),
					RequestSize:   req.Header.Get(echo.HeaderContentLength),
					ResponseSize:  strconv.FormatInt(size, 10),
					Latency:       strconv.FormatFloat(stop.Sub(start).Seconds(), 'f', 9, 64) + "s",
				}))
			return nil
		}
	}
}
