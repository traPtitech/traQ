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
	rbac.Add(Admin)
	rbac.Add(User)
	rbac.Add(Bot)

	// 一般ユーザーのパーミッション
	for _, p := range []gorbac.Permission{
		permission.CreateChannels,
		permission.GetChannel,
		permission.GetChannels,
		permission.PatchChannel,
	} {
		User.Permit(p)
	}

	// 管理者ユーザーのパーミッション
	// ※一般ユーザーのパーミッションを全て含む
	rbac.SetParent(Admin.ID(), User.ID())
	for _, p := range []gorbac.Permission{
		permission.DeleteChannel,
	} {
		Admin.Permit(p)
	}
}
