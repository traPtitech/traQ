package role

import "github.com/mikespook/gorbac"

var (
	// Admin : 管理者ロール
	Admin = gorbac.NewStdRole("admin")
	// User : 一般ユーザーロール
	User = gorbac.NewStdRole("user")
	// Bot : Botユーザーロール
	Bot = gorbac.NewStdRole("bot")
)
