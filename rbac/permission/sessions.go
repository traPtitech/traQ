package permission

import (
	"github.com/traPtitech/traQ/rbac"
)

const (
	// GetMySessions セッションリスト取得権限
	GetMySessions = rbac.Permission("get_my_sessions")
	// DeleteMySessions セッション削除権限
	DeleteMySessions = rbac.Permission("delete_my_sessions")
)
