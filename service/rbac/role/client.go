package role

import (
	"github.com/traPtitech/traQ/service/rbac/permission"
)

// Cliet Clientロール (for OAuth2 client credentials grant)
const Client = "client"

var clientPerms = []permission.Permission{
	permission.GetUser,
	permission.GetUserTag,
	permission.GetUserGroup,
	permission.GetStamp,
}
