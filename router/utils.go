package router

import (
	"github.com/traPtitech/traQ/bot"
	"github.com/traPtitech/traQ/oauth2"
	"net/http"

	"github.com/labstack/echo"
)

var errMySQLDuplicatedRecord uint16 = 1062

// Handlers ハンドラ
type Handlers struct {
	Bot    *bot.Dao
	OAuth2 oauth2.Store
}

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
		c.Echo().Logger.Errorf("an error occurred while sending to JSON: %v", err)
	}

}
