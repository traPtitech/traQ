package permission

import "github.com/mikespook/gorbac"

var (
	// GetTag : ユーザータグ取得権限
	GetTag = gorbac.NewStdPermission("get_tag")
	// AddTag : ユーザータグ追加権限
	AddTag = gorbac.NewStdPermission("add_tag")
	// RemoveTag : ユーザータグ削除権限
	RemoveTag = gorbac.NewStdPermission("remove_tag")
	// ChangeTagLockState : ユーザータグロック状態変更権限
	ChangeTagLockState = gorbac.NewStdPermission("change_tag_lock_state")
)
