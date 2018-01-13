package router

import (
	"net/http"

	"github.com/labstack/echo"
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
		c.Echo().Logger.Errorf("an error occured while sending to JSON: %v", err)
	}

}
