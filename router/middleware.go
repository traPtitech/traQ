package router

import (
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/sessions"
	"net/http"

	"github.com/traPtitech/traQ/oauth2"

	"fmt"

	"github.com/labstack/echo"
	"github.com/mikespook/gorbac"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/rbac"
)

// UserAuthenticate User認証するミドルウェア
func (h *Handlers) UserAuthenticate(oh *oauth2.Handler) echo.MiddlewareFunc {
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
				user, err := h.Repo.GetUser(token.UserID)
				if err != nil {
					switch err {
					case repository.ErrNotFound:
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
				sess, err := sessions.Get(c.Response(), c.Request(), false)
				if err != nil {
					c.Logger().Errorf("Failed to get a session: %v", err)
					return echo.NewHTTPError(http.StatusInternalServerError)
				}
				if sess == nil || sess.GetUserID() == uuid.Nil {
					return echo.NewHTTPError(http.StatusForbidden, "You are not logged in")
				}

				user, err := h.Repo.GetUser(sess.GetUserID())
				if err != nil {
					switch err {
					case repository.ErrNotFound:
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
					if !rbac.IsGranted(user.ID, user.Role, v) {
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

// RequestBodyLengthLimit リクエストボディのContentLengthで制限をかけるミドルウェア
func RequestBodyLengthLimit(kb int64) echo.MiddlewareFunc {
	limit := kb << 10
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if l := c.Request().Header.Get(echo.HeaderContentLength); len(l) == 0 {
				return echo.NewHTTPError(http.StatusLengthRequired) // ContentLengthを送ってこないリクエストを殺す
			}
			if c.Request().ContentLength > limit {
				return echo.NewHTTPError(http.StatusRequestEntityTooLarge, fmt.Sprintf("the request must be smaller than %dKB", kb))
			}
			return next(c)
		}
	}
}

// ValidateGroupID 'groupID'パラメータのグループを検証するミドルウェア
func (h *Handlers) ValidateGroupID(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		groupID := getRequestParamAsUUID(c, paramGroupID)

		g, err := h.Repo.GetUserGroup(groupID)
		if err != nil {
			switch err {
			case repository.ErrNotFound:
				return c.NoContent(http.StatusNotFound)
			default:
				c.Logger().Error(err)
				return c.NoContent(http.StatusInternalServerError)
			}
		}
		c.Set("paramGroup", g)

		return next(c)
	}
}
