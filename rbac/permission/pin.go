package permission

import (
	"github.com/traPtitech/traQ/rbac"
)

const (
	// GetPin : ピン留め取得権限
	GetPin = rbac.Permission("get_pin")
	// CreatePin : ピン留め作成権限
	CreatePin = rbac.Permission("create_pin")
	// DeletePin : ピン留め削除権限
	DeletePin = rbac.Permission("delete_pin")
)
