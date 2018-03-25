package rbac

import (
	"github.com/mikespook/gorbac"
	"github.com/satori/go.uuid"
)

// Store : RBACのルール永続化用ストアインターフェイス
type Store interface {
	OverrideStore
}

// OverrideStore : RBACのオーバーライドルールの永続化用ストアインターフェイス
type OverrideStore interface {
	GetAllOverrides() ([]OverrideData, error)
	SaveOverride(userID uuid.UUID, p gorbac.Permission, validity bool) error
	DeleteOverride(userID uuid.UUID, p gorbac.Permission) error
}
