package router

import (
	"net/http"

	"github.com/labstack/echo"
	"github.com/labstack/echo-contrib/session"
	"github.com/traPtitech/traQ/model"
)

// GetUserInfo User情報を取得するミドルウェア
func GetUserInfo(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		sess, err := session.Get("sessions", c)
		if err != nil {
			c.Echo().Logger.Errorf("Failed to get a session: %v", err)
			return echo.NewHTTPError(http.StatusForbidden, "your ID isn't found")
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
			return echo.NewHTTPError(http.StatusInternalServerError, "cannnot get your user infomation")
		}
		c.Set("user", user)
		return next(c)
	}
}
