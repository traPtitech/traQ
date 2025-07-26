package middlewares

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/router/consts"
	"golang.org/x/time/rate"
)

// RateLimit 各ユーザーについて、すべてのエンドポイントでのリクエスト数を制限するミドルウェア
func RateLimit(limit rate.Limit) echo.MiddlewareFunc {
	return middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Store: middleware.NewRateLimiterMemoryStore(limit),
		IdentifierExtractor: func(ctx echo.Context) (string, error) {
			user := ctx.Get(consts.KeyUser).(model.UserInfo)
			ip := ctx.RealIP()
			return user.GetID().String() + ip, nil
		},
	})
}
