package role

import (
	"github.com/mikespook/gorbac"
	"github.com/traPtitech/traQ/rbac"
	"github.com/traPtitech/traQ/rbac/permission"
)

var (
	// Admin : 管理者ユーザーロール
	Admin = gorbac.NewStdRole("admin")
	// User : 一般ユーザーロール
	User = gorbac.NewStdRole("user")
	// Bot : Botユーザーロール
	Bot = gorbac.NewStdRole("bot")
)

// SetRole : rbacに既定のロールをセットします
func SetRole(rbac *rbac.RBAC) {
	// 一般ユーザーのパーミッション
	for _, p := range []gorbac.Permission{
		permission.CreateChannels,
		permission.GetChannel,
		permission.GetChannels,
		permission.PatchChannel,
	} {
		if err := User.Assign(p); err != nil {
			panic(err)
		}
	}

	// 管理者ユーザーのパーミッション
	// ※一般ユーザーのパーミッションを全て含む
	for _, p := range []gorbac.Permission{
		permission.DeleteChannel,
	} {
		if err := Admin.Assign(p); err != nil {
			panic(err)
		}
	}

	for _, r := range []gorbac.Role{
		Bot,
		User,
		Admin,
	} {
		if err := rbac.Add(r); err != nil {
			panic(err)
		}
	}
	if err := rbac.SetParent(Admin.ID(), User.ID()); err != nil {
		panic(err)
	}
}
