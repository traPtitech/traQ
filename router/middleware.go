package router

import (
	"net/http"

	"fmt"
	"github.com/labstack/echo"
	"github.com/labstack/echo-contrib/session"
	"github.com/mikespook/gorbac"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/rbac"
)

// GetUserInfo User情報を取得するミドルウェア
func GetUserInfo(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		sess, err := session.Get("sessions", c)
		if err != nil {
			c.Echo().Logger.Errorf("Failed to get a session: %v", err)
			return echo.NewHTTPError(http.StatusForbidden, "Your userID isn't found")
		}
		var userID string
		if sess.Values["userID"] != nil {
			userID = sess.Values["userID"].(string)
		} else {
			c.Echo().Logger.Errorf("This session doesn't have a userID")
			return echo.NewHTTPError(http.StatusForbidden, "Your userID doesn't exist")
		}

		user, err := model.GetUser(userID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Cannot get your user information")
		}
		c.Set("user", user)
		c.Set("userRole", user.Role)
		c.Set("userID", userID)
		return next(c)
	}
}

// AccessControlMiddlewareGenerator アクセスコントロールミドルウェアのジェネレーターを返します
func AccessControlMiddlewareGenerator(rbac *rbac.RBAC) func(p ...gorbac.Permission) echo.MiddlewareFunc {
	return func(p ...gorbac.Permission) echo.MiddlewareFunc {
		return func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
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
