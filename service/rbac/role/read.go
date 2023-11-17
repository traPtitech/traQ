package role

import (
	"github.com/traPtitech/traQ/service/rbac/permission"
)

// Read 読み取り専用ユーザーロール
const Read = "read"

var readPerms = []permission.Permission{
	permission.GetChannel,
	permission.GetMessage,
	permission.GetChannelSubscription,
	permission.ConnectNotificationStream,
	permission.GetUser,
	permission.GetMe,
	permission.GetOIDCUserInfo,
	permission.GetChannelStar,
	permission.GetUnread,
	permission.GetUserTag,
	permission.GetUserGroup,
	permission.GetStamp,
	permission.GetMyStampHistory,
	permission.DownloadFile,
	permission.GetWebhook,
	permission.GetBot,
	permission.GetClipFolder,
	permission.GetStampPalette,
}
