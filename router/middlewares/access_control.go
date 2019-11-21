package middlewares

import (
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/rbac"
	"github.com/traPtitech/traQ/rbac/role"
	"github.com/traPtitech/traQ/router/consts"
	"net/http"
)

// AccessControlMiddlewareGenerator アクセスコントロールミドルウェアのジェネレーターを返します
func AccessControlMiddlewareGenerator(r rbac.RBAC) func(p ...rbac.Permission) echo.MiddlewareFunc {
	return func(p ...rbac.Permission) echo.MiddlewareFunc {
		return func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				// OAuth2スコープ権限検証
				if scopes, ok := c.Get(consts.KeyOAuth2AccessScopes).(model.AccessScopes); ok {
					for _, v := range p {
						if !r.IsAnyGranted(scopes.StringArray(), v) {
							// NG
							return echo.NewHTTPError(http.StatusForbidden, fmt.Sprintf("you are not permitted to request to '%s'", c.Request().URL.Path))
						}
					}
				}

				// ユーザー権限検証
				user := c.Get(consts.KeyUser).(*model.User)
				for _, v := range p {
					if !r.IsGranted(user.Role, v) {
						// NG
						return echo.NewHTTPError(http.StatusForbidden, fmt.Sprintf("you are not permitted to request to '%s'", c.Request().URL.Path))
					}
				}

				return next(c) // OK
			}
		}
	}
}

// AdminOnly 管理者ユーザーのみを通すミドルウェア
func AdminOnly(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// ユーザーロール検証
		user := c.Get(consts.KeyUser).(*model.User)
		if user.Role != role.Admin {
			return echo.NewHTTPError(http.StatusForbidden, fmt.Sprintf("you are not permitted to request to '%s'", c.Request().URL.Path))
		}
		return next(c) // OK
	}
}
