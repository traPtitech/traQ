package router

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo"
	"github.com/labstack/echo-contrib/session"
)

// CustomHTTPErrorHandler :json形式でエラーレスポンスを返す
func CustomHTTPErrorHandler(err error, c echo.Context) {
	var (
		code = http.StatusInternalServerError
		msg  interface{}
	)

	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
		msg = he.Message
	} else {
		msg = http.StatusText(code)
	}
	if _, ok := msg.(string); ok {
		msg = map[string]interface{}{"message": msg}
	}

	if err = c.JSON(code, msg); err != nil {
		//TODO: log出力
	}

}

func getUserID(c echo.Context) (string, error) {
	sess, err := session.Get("sessions", c)
	if err != nil {
		fmt.Errorf("Failed to get a session: %v", err)
		return "", echo.NewHTTPError(http.StatusForbidden, "your id isn't found")

	}

	var userID string
	if sess.Values["userId"] != nil {
		userID = sess.Values["userId"].(string)
	} else {
		fmt.Errorf("This session doesn't have a userId")
		return "", echo.NewHTTPError(http.StatusForbidden, "Your userID doesn't exist")
	}
	return userID, nil
}
