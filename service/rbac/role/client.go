package role

import (
	"github.com/traPtitech/traQ/service/rbac/permission"
)

// Client Clientロール (for OAuth2 client credentials grant)
const Client = "client"

// 自分自身以外の参照系は許可するようにしたいが、https://github.com/traPtitech/traQ/pull/2433#discussion_r1649383346 
// の事情から許可できる権限が限られる
// https://github.com/traPtitech/traQ/issues/2463 で権限を増やせるよう対応予定
var clientPerms = []permission.Permission{
	permission.GetUser,
	permission.GetUserTag,
	permission.GetUserGroup,
	permission.GetStamp,
}
