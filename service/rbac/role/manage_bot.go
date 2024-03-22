package role

import (
	"github.com/traPtitech/traQ/service/rbac/permission"
)

// ManageBot BotConsoleロール
const ManageBot = "manage_bot"

var manageBotPerms = []permission.Permission{
	permission.GetChannel,
	permission.GetUser,
	permission.GetMe,
	permission.GetOIDCUserInfo,
	permission.GetWebhook,
	permission.CreateWebhook,
	permission.EditWebhook,
	permission.DeleteWebhook,
	permission.GetBot,
	permission.CreateBot,
	permission.EditBot,
	permission.DeleteBot,
	permission.BotActionJoinChannel,
	permission.BotActionLeaveChannel,
	permission.GetClients,
	permission.CreateClient,
	permission.EditMyClient,
	permission.DeleteMyClient,
}
