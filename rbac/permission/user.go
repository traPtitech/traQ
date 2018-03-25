package permission

import "github.com/mikespook/gorbac"

var (
	// GetUser : ユーザー情報取得権限
	GetUser = gorbac.NewStdPermission("get_user")
	// RegisterUser : 新規ユーザー登録権限
	RegisterUser = gorbac.NewStdPermission("register_user")
	// GetMe : 自ユーザー情報取得権限
	GetMe = gorbac.NewStdPermission("get_me")
	// EditMe : 自ユーザー情報変更権限
	EditMe = gorbac.NewStdPermission("edit_me")
	// ChangeMyIcon : 自ユーザーアイコン変更権限
	ChangeMyIcon = gorbac.NewStdPermission("change_my_icon")
)
