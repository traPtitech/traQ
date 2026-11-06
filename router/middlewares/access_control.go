package middlewares

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v5"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/service/rbac"
	"github.com/traPtitech/traQ/service/rbac/permission"
	"github.com/traPtitech/traQ/service/rbac/role"
)

// AccessControlMiddlewareGenerator アクセスコントロールミドルウェアのジェネレーターを返します
func AccessControlMiddlewareGenerator(r rbac.RBAC) func(p ...permission.Permission) echo.MiddlewareFunc {
	return func(p ...permission.Permission) echo.MiddlewareFunc {
		return func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c *echo.Context) error {
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
				user := c.Get(consts.KeyUser).(model.UserInfo)
				if user == nil {
					for _, v := range p {
						if !r.IsGranted(role.Client, v) {
							// NG
							return echo.NewHTTPError(http.StatusForbidden, fmt.Sprintf("you are not permitted to request to '%s'", c.Request().URL.Path))
						}
					}

					return next(c) // OK
				}
				for _, v := range p {
					if !r.IsGranted(user.GetRole(), v) {
						// NG
						return echo.NewHTTPError(http.StatusForbidden, fmt.Sprintf("you are not permitted to request to '%s'", c.Request().URL.Path))
					}
				}

				return next(c) // OK
			}
		}
	}
}

// BlockBot Botのリクエストを制限するミドルウェア
func BlockBot() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			user := c.Get(consts.KeyUser).(model.UserInfo)
			if user.IsBot() {
				return herror.Forbidden("Bot users are not permitted to access this API")
			}
			return next(c)
		}
	}
}

// BlockNonBot Bot以外のリクエストを制限するミドルウェア
func BlockNonBot() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			user := c.Get(consts.KeyUser).(model.UserInfo)
			if !user.IsBot() {
				return herror.Forbidden("Non-bot users are not permitted to access this API")
			}
			return next(c)
		}
	}
}

// CheckClientAccessPerm Clientアクセス権限を確認するミドルウェア
func CheckClientAccessPerm(rbac rbac.RBAC) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			user := c.Get(consts.KeyUser).(model.UserInfo)
			oc := c.Get(consts.KeyParamClient).(*model.OAuth2Client)

			// アクセス権確認
			if !rbac.IsGranted(user.GetRole(), permission.ManageOthersClient) && oc.CreatorID != user.GetID() {
				return herror.Forbidden()
			}

			return next(c)
		}
	}
}

// CheckDeleteStampPerm スタンプ削除権限を確認するミドルウェア
func CheckDeleteStampPerm(rbac rbac.RBAC) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			user := c.Get(consts.KeyUser).(model.UserInfo)

			if rbac.IsGranted(user.GetRole(), permission.DeleteStamp) {
				return next(c) //adminのアクセス
			}
			stamp := c.Get(consts.KeyParamStamp).(*model.Stamp)
			if rbac.IsGranted(user.GetRole(), permission.DeleteMyStamp) && stamp.CreatorID == user.GetID() {
				return next(c) //スタンプ作成者のアクセス
			}
			return herror.Forbidden()
		}
	}
}
