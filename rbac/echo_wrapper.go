package rbac

import (
	"fmt"
	"github.com/labstack/echo"
	"github.com/mikespook/gorbac"
	"github.com/satori/go.uuid"
	"net/http"
)

// HandleWithRBAC : リクエストユーザーが指定したパーミッションを持っていない場合に403エラーを返すハンドラを生成します
func (rbac *RBAC) HandleWithRBAC(p gorbac.Permission, h echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		userID := c.Get("userID").(string)
		role := c.Get("userRole").(string)

		if rbac.IsGranted(uuid.FromStringOrNil(userID), role, p) {
			return h(c) // OK
		}

		// NG
		return echo.NewHTTPError(http.StatusForbidden, fmt.Sprintf("you are not permitted to request to '%s'", c.Request().URL.Path))
	}
}
