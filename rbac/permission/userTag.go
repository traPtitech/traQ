package permission

import (
	"github.com/traPtitech/traQ/rbac"
)

const (
	// GetTag : ユーザータグ取得権限
	GetTag = rbac.Permission("get_tag")
	// AddTag : ユーザータグ追加権限
	AddTag = rbac.Permission("add_tag")
	// RemoveTag : ユーザータグ削除権限
	RemoveTag = rbac.Permission("remove_tag")
	// ChangeTagLockState : ユーザータグロック状態変更権限
	ChangeTagLockState = rbac.Permission("change_tag_lock_state")
	// EditTag : タグ情報編集権限
	EditTag = rbac.Permission("edit_tag")
)
