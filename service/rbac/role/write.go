package role

import (
	"github.com/traPtitech/traQ/service/rbac/permission"
)

// Write 書き込み専用ユーザーロール
const Write = "write"

var writePerms = []permission.Permission{
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
	permission.DeleteFile,
	permission.CreateClipFolder,
	permission.EditClipFolder,
	permission.DeleteClipFolder,
	permission.CreateStampPalette,
	permission.EditStampPalette,
	permission.DeleteStampPalette,
}
