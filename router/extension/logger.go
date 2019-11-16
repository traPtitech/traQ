package extension

import (
	"fmt"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/router/consts"
)

// GetTraceID トレースIDを返します
func GetTraceID(c echo.Context) string {
	v, ok := c.Get(consts.KeyTraceID).(string)
	if ok {
		return v
	}
	v = fmt.Sprintf("%02x", uuid.Must(uuid.NewV4()).Bytes())
	c.Set(consts.KeyTraceID, v)
	return v
}
