package router

import (
	"net/http"

	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/model"
)

// GetUserInfo User情報を取得するミドルウェア
func GetUserInfo(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		userID, err := getUserID(c)
		if err != nil {
			return echo.NewHTTPError(http.StatusForbidden, "your id is not found")
		}

		user, err := model.GetUser(userID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "cannnot get your user infomation")
		}
		c.Set("userID", user)
		return next(c)
	}
}
