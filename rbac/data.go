package rbac

import (
	"github.com/gofrs/uuid"
	"github.com/mikespook/gorbac"
)

// OverrideData : RBACのオーバライドルールのインターフェイス
type OverrideData interface {
	GetUserID() uuid.UUID
	GetPermission() gorbac.Permission
	GetValidity() bool
}
