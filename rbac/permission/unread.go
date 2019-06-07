package permission

import (
	"github.com/traPtitech/traQ/rbac"
)

const (
	// GetUnread : 未読メッセージ一覧の取得権限
	GetUnread = rbac.Permission("get_unread")
	// DeleteUnread : メッセージ既読化権限
	DeleteUnread = rbac.Permission("delete_unread")
)
