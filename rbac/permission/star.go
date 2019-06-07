package permission

import (
	"github.com/traPtitech/traQ/rbac"
)

const (
	// GetStar : スター取得権限
	GetStar = rbac.Permission("get_star")
	// CreateStar : スター作成権限
	CreateStar = rbac.Permission("create_star")
	// DeleteStar : スター削除権限
	DeleteStar = rbac.Permission("delete_star")
)
