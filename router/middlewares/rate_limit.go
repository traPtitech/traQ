package middlewares

import (
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/router/consts"
	"golang.org/x/time/rate"
)

// RateLimit 各ユーザーについて、すべてのエンドポイントでのリクエスト数を制限するミドルウェア
func RateLimit(limit rate.Limit, burst int, expiresIn time.Duration) echo.MiddlewareFunc {
	return middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(middleware.RateLimiterMemoryStoreConfig{
			Rate:      limit,
			Burst:     burst,
			ExpiresIn: expiresIn,
		}),
		IdentifierExtractor: func(ctx echo.Context) (string, error) {
			user := ctx.Get(consts.KeyUser).(model.UserInfo)
			ip := ctx.RealIP()
			return user.GetID().String() + ip, nil
		},
	})
}
