package role

import (
	"github.com/traPtitech/traQ/service/rbac/permission"
)

// Qall Qall(通話)利用ロール (for OAuth2)
const Qall = "qall"

var qallPerms = []permission.Permission{
	permission.WebRTC,
}
