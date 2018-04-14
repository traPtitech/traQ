package router

import (
	"net/http"

	"github.com/traPtitech/traQ/oauth2"

	"fmt"

	"github.com/labstack/echo"
	"github.com/labstack/echo-contrib/session"
	"github.com/mikespook/gorbac"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/rbac"
)

// UserAuthenticate User認証するミドルウェア
func UserAuthenticate(oh *oauth2.Handler) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ah := c.Request().Header.Get(echo.HeaderAuthorization)
			if len(ah) > 0 {
				// AuthorizationヘッダーがあるためOAuth2で検証

				// Authorizationスキーム検証
				l := len(oauth2.AuthScheme)
				if !(len(ah) > l+1 && ah[:l] == oauth2.AuthScheme) {
					return echo.NewHTTPError(http.StatusForbidden, "the Authorization Header's scheme is invalid")
				}

				// OAuth2 Token検証
				token, err := oh.GetTokenByAccess(ah[l+1:])
				if err != nil {
					switch err {
					case oauth2.ErrTokenNotFound:
						return echo.NewHTTPError(http.StatusForbidden, "the token is invalid")
					default:
						c.Logger().Error(err)
						return echo.NewHTTPError(http.StatusInternalServerError)
					}
				}

				// tokenの有効期限の検証
				if token.IsExpired() {
					return echo.NewHTTPError(http.StatusForbidden, "the token is expired")
				}

				// tokenの検証に成功。ユーザーを取得
				user, err := model.GetUser(token.UserID.String())
				if err != nil {
					switch err {
					case model.ErrNotFound:
						return echo.NewHTTPError(http.StatusForbidden, "the user is not found")
					default:
						c.Logger().Error(err)
						return echo.NewHTTPError(http.StatusInternalServerError)
					}
				}

				c.Set("user", user)
				c.Set("userID", user.ID)
				// 認可に基づきRole生成
				c.Set("role", token.Scopes.GenerateRole())
			} else {
				// Authorizationヘッダーがないためセッションを確認する
				sess, err := session.Get("sessions", c)
				if err != nil {
					c.Logger().Errorf("Failed to get a session: %v", err)
					return echo.NewHTTPError(http.StatusForbidden, "You are not logged in")
				}
				if sess.Values["userID"] == nil {
					return echo.NewHTTPError(http.StatusForbidden, "You are not logged in")
				}

				user, err := model.GetUser(sess.Values["userID"].(string))
				if err != nil {
					switch err {
					case model.ErrNotFound:
						return echo.NewHTTPError(http.StatusForbidden, "the user is not found")
					default:
						c.Logger().Error(err)
						return echo.NewHTTPError(http.StatusInternalServerError)
					}
				}

				c.Set("user", user)
				c.Set("userID", user.ID)
			}
			return next(c)
		}
	}
}

// AccessControlMiddlewareGenerator アクセスコントロールミドルウェアのジェネレーターを返します
func AccessControlMiddlewareGenerator(rbac *rbac.RBAC) func(p ...gorbac.Permission) echo.MiddlewareFunc {
	return func(p ...gorbac.Permission) echo.MiddlewareFunc {
		return func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				// クライアント権限検証
				if role, ok := c.Get("role").(gorbac.Role); ok {
					for _, v := range p {
						if !role.Permit(v) {
							// NG
							return echo.NewHTTPError(http.StatusForbidden, fmt.Sprintf("you are not permitted to request to '%s'", c.Request().URL.Path))
						}
					}
				}

				// ユーザー権限検証
				user := c.Get("user").(*model.User)
				for _, v := range p {
					if !rbac.IsGranted(uuid.FromStringOrNil(user.ID), user.Role, v) {
						// NG
						return echo.NewHTTPError(http.StatusForbidden, fmt.Sprintf("you are not permitted to request to '%s'", c.Request().URL.Path))
					}
				}
				c.Set("rbac", rbac)

				return next(c) // OK
			}
		}
	}
}
