package rbac

import (
	"github.com/mikespook/gorbac"
	"github.com/satori/go.uuid"
)

// OverrideData : RBACのオーバライドルールのインターフェイス
type OverrideData interface {
	GetUserID() uuid.UUID
	GetPermission() gorbac.Permission
	GetValidity() bool
}
