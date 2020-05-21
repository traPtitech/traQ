package middlewares

import (
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/router/session"
)

// NoLogin セッションが既に存在するリクエストを拒否するミドルウェア
func NoLogin(sessStore session.Store) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if len(c.Request().Header.Get(echo.HeaderAuthorization)) > 0 {
				return herror.BadRequest("Authorization Header must not be set. Please logout once.")
			}

			sess, err := sessStore.GetSession(c, false)
			if err != nil {
				return herror.InternalServerError(err)
			}
			if sess != nil && sess.LoggedIn() {
				return herror.BadRequest("You have already logged in. Please logout once.")
			}

			return next(c)
		}
	}
}
