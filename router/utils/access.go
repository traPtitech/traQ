package utils

import (
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v5"

	"github.com/traPtitech/traQ/router/extension/herror"
)

// ParseUUIDParam URLのnameパラメータをUUIDとしてパースする
//
// 不正な値やNilの場合は、そのパスのリソースが存在しないものとして404を返す。
func ParseUUIDParam(c *echo.Context, name string) (uuid.UUID, error) {
	id, err := uuid.FromString(c.Param(name))
	if err != nil || id == uuid.Nil {
		return uuid.Nil, herror.NotFound()
	}
	return id, nil
}
