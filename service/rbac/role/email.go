package role

import (
	"github.com/traPtitech/traQ/service/rbac/permission"
)

// Email ユーザー情報読み取り専用ロール (for OIDC)
const Email = "email"

var emailPerms = []permission.Permission{
	permission.GetOIDCUserInfo,
}
