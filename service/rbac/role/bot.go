package role

import (
	"github.com/traPtitech/traQ/service/rbac/permission"
)

// Bot Botユーザーロール
const Bot = "bot"

var botPerms = []permission.Permission{
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
	permission.GetOIDCUserInfo,
	permission.EditMe,
	permission.GetMyStampHistory,
	permission.GetMyStampRecommendations,
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
	permission.DeleteFile,
	permission.BotActionJoinChannel,
	permission.BotActionLeaveChannel,
	permission.WebRTC,
}
