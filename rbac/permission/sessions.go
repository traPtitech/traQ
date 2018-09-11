package permission

import "github.com/mikespook/gorbac"

var (
	// GetMySessions セッションリスト取得権限
	GetMySessions = gorbac.NewStdPermission("get_my_sessions")
	// DeleteMySessions セッション削除権限
	DeleteMySessions = gorbac.NewStdPermission("delete_my_sessions")
)
