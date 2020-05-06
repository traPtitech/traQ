package extension

import (
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/utils/random"
)

// GetRequestID リクエストIDを返します
func GetRequestID(c echo.Context) string {
	rid := c.Request().Header.Get(echo.HeaderXRequestID)
	if len(rid) == 0 {
		rid = random.AlphaNumeric(32)
	}
	return rid
}
