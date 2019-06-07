package role

import (
	"github.com/traPtitech/traQ/rbac"
	"github.com/traPtitech/traQ/rbac/permission"
)

// Bot Botユーザーロール
const Bot = "bot"

var botPerms = []rbac.Permission{
	permission.GetChannel,
	permission.GetTopic,
	permission.EditTopic,
	permission.GetMessage,
	permission.PostMessage,
	permission.EditMessage,
	permission.DeleteMessage,
	permission.GetPin,
	permission.CreatePin,
	permission.DeletePin,
	permission.GetNotificationStatus,
	permission.ChangeNotificationStatus,
	permission.GetUser,
	permission.GetMe,
	permission.GetTag,
	permission.AddTag,
	permission.RemoveTag,
	permission.ChangeTagLockState,
	permission.GetStamp,
	permission.GetMessageStamp,
	permission.AddMessageStamp,
	permission.RemoveMessageStamp,
	permission.DownloadFile,
}
