package middlewares

import (
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/router/sessions"
)

// NoLogin セッションが既に存在するリクエストを拒否するミドルウェア
func NoLogin() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if len(c.Request().Header.Get(echo.HeaderAuthorization)) > 0 {
				return herror.BadRequest("Authorization Header must not be set. Please logout once.")
			}

			sess, err := sessions.Get(c.Response(), c.Request(), false)
			if err != nil {
				return herror.InternalServerError(err)
			}
			if sess != nil {
				if sess.GetUserID() != uuid.Nil {
					return herror.BadRequest("You have already logged in. Please logout once.")
				}
				_ = sess.Destroy(c.Response(), c.Request())
			}

			return next(c)
		}
	}
}
