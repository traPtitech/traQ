package role

import (
	"github.com/traPtitech/traQ/rbac"
	"github.com/traPtitech/traQ/rbac/permission"
)

// Write 書き込み専用ユーザーロール
const Write = "write"

var writePerms = []rbac.Permission{
	permission.CreateChannel,
	permission.EditTopic,
	permission.PostMessage,
	permission.EditMessage,
	permission.DeleteMessage,
	permission.ReportMessage,
	permission.CreatePin,
	permission.DeletePin,
	permission.ChangeNotificationStatus,
	permission.RegisterDevice,
	permission.EditMe,
	permission.ChangeMyIcon,
	permission.CreateClip,
	permission.DeleteClip,
	permission.CreateClipFolder,
	permission.PatchClipFolder,
	permission.DeleteClipFolder,
	permission.CreateStar,
	permission.DeleteStar,
	permission.DeleteUnread,
	permission.MuteChannel,
	permission.UnmuteChannel,
	permission.AddTag,
	permission.RemoveTag,
	permission.ChangeTagLockState,
	permission.CreateStamp,
	permission.AddMessageStamp,
	permission.RemoveMessageStamp,
	permission.EditStamp,
	permission.UploadFile,
	permission.PostHeartbeat,
	permission.InstallBot,
	permission.UninstallBot,
}
