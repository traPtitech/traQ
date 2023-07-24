package role

import (
	"github.com/traPtitech/traQ/service/rbac/permission"
)

// Profile ユーザー情報読み取り専用ロール
const Profile = "profile"

var profilePerms = []permission.Permission{
	permission.GetMe,
}
