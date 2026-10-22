package extension

import (
	"github.com/labstack/echo/v5"

	"github.com/traPtitech/traQ/utils/random"
)

const requestIDKey = "traq_request_id"

// GetRequestID リクエストIDを返します
func GetRequestID(c *echo.Context) string {
	if rid, ok := c.Get(requestIDKey).(string); ok {
		return rid
	}
	rid := c.Request().Header.Get(echo.HeaderXRequestID)
	if len(rid) == 0 {
		rid = random.AlphaNumeric(32)
	}
	c.Set(requestIDKey, rid)
	return rid
}
