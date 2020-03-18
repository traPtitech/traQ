package role

import (
	"github.com/traPtitech/traQ/rbac"
	"github.com/traPtitech/traQ/rbac/permission"
)

// Write 書き込み専用ユーザーロール
const Write = "write"

var writePerms = []rbac.Permission{
	permission.CreateChannel,
	permission.EditChannelTopic,
	permission.PostMessage,
	permission.EditMessage,
	permission.DeleteMessage,
	permission.ReportMessage,
	permission.CreateMessagePin,
	permission.DeleteMessagePin,
	permission.EditChannelSubscription,
	permission.RegisterFCMDevice,
	permission.EditMe,
	permission.ChangeMyIcon,
	permission.EditChannelStar,
	permission.DeleteUnread,
	permission.EditUserTag,
	permission.CreateUserGroup,
	permission.EditUserGroup,
	permission.DeleteUserGroup,
	permission.CreateStamp,
	permission.AddMessageStamp,
	permission.RemoveMessageStamp,
	permission.EditStamp,
	permission.UploadFile,
	permission.PostHeartbeat,
	permission.CreateClipFolder,
	permission.EditClipFolder,
	permission.DeleteClipFolder,
	permission.CreateStampPalette,
	permission.EditStampPalette,
	permission.DeleteStampPalette,
}
