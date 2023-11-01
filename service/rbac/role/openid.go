package role

import (
	"github.com/traPtitech/traQ/service/rbac/permission"
)

// OpenID OIDC専用ロール
const OpenID = "openid"

var openIDPerms []permission.Permission
