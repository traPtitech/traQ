package role

import (
	"github.com/traPtitech/traQ/rbac"
	"github.com/traPtitech/traQ/rbac/permission"
)

// Bot Botユーザーロール
const Bot = "bot"

var botPerms = []rbac.Permission{
	permission.GetChannel,
	permission.EditChannelTopic,
	permission.GetMessage,
	permission.PostMessage,
	permission.EditMessage,
	permission.DeleteMessage,
	permission.CreateMessagePin,
	permission.DeleteMessagePin,
	permission.GetChannelSubscription,
	permission.EditChannelSubscription,
	permission.GetUser,
	permission.GetMe,
	permission.EditMe,
	permission.GetMyStampHistory,
	permission.ChangeMyIcon,
	permission.GetUserTag,
	permission.EditUserTag,
	permission.GetUserGroup,
	permission.CreateUserGroup,
	permission.EditUserGroup,
	permission.DeleteUserGroup,
	permission.GetStamp,
	permission.AddMessageStamp,
	permission.RemoveMessageStamp,
	permission.DownloadFile,
	permission.UploadFile,
	permission.BotActionJoinChannel,
	permission.BotActionLeaveChannel,
}
