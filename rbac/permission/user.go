package permission

import (
	"github.com/traPtitech/traQ/rbac"
)

const (
	// GetUser ユーザー情報取得権限
	GetUser = rbac.Permission("get_user")
	// RegisterUser 新規ユーザー登録権限
	RegisterUser = rbac.Permission("register_user")
	// GetMe 自ユーザー情報取得権限
	GetMe = rbac.Permission("get_me")
	// EditMe 自ユーザー情報変更権限
	EditMe = rbac.Permission("edit_me")
	// ChangeMyIcon 自ユーザーアイコン変更権限
	ChangeMyIcon = rbac.Permission("change_my_icon")
	// ChangeMyPassword 自ユーザーパスワード変更権限
	ChangeMyPassword = rbac.Permission("change_my_password")
	// EditOtherUsers 他ユーザー情報変更権限
	EditOtherUsers = rbac.Permission("edit_other_users")
)
