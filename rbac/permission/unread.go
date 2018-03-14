package permission

import "github.com/mikespook/gorbac"

var (
	// GetUnread : 未読メッセージ一覧の取得権限
	GetUnread = gorbac.NewStdPermission("get_unread")
	// DeleteUnread : メッセージ既読化権限
	DeleteUnread = gorbac.NewStdPermission("delete_unread")
)
