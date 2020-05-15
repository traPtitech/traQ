package role

import (
	"github.com/traPtitech/traQ/service/rbac/permission"
)

// User 一般ユーザーロール
const User = "user"

var userPerms = []permission.Permission{
	// read, writeロールのパーミッションを全て含む
	permission.ChangeMyPassword,
	permission.GetUserQRCode,
	permission.GetMySessions,
	permission.DeleteMySessions,
	permission.GetMyTokens,
	permission.RevokeMyToken,
	permission.GetMyExternalAccount,
	permission.EditMyExternalAccount,
	permission.GetClients,
	permission.CreateClient,
	permission.EditMyClient,
	permission.DeleteMyClient,
	permission.CreateWebhook,
	permission.EditWebhook,
	permission.DeleteWebhook,
	permission.CreateBot,
	permission.EditBot,
	permission.DeleteBot,
	permission.BotActionJoinChannel,
	permission.BotActionLeaveChannel,
	permission.WebRTC,
}

func init() {
	userPerms = append(userPerms, readPerms...)
	userPerms = append(userPerms, writePerms...)
}
