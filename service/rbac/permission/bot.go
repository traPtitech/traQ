package permission

import (
	"github.com/traPtitech/traQ/service/rbac"
)

const (
	// GetWebhook Webhook情報取得権限
	GetWebhook = rbac.Permission("get_webhook")
	// CreateWebhook Webhook作成権限
	CreateWebhook = rbac.Permission("create_webhook")
	// EditWebhook Webhook編集権限
	EditWebhook = rbac.Permission("edit_webhook")
	// DeleteWebhook Webhook削除権限
	DeleteWebhook = rbac.Permission("delete_webhook")
	// AccessOthersWebhook 他人のWebhookのアクセス権限
	AccessOthersWebhook = rbac.Permission("access_others_webhook")

	// GetBot Bot情報取得権限
	GetBot = rbac.Permission("get_bot")
	// CreateBot Bot作成権限
	CreateBot = rbac.Permission("create_bot")
	// EditBot Bot編集権限
	EditBot = rbac.Permission("edit_bot")
	// DeleteBot Bot削除権限
	DeleteBot = rbac.Permission("delete_bot")
	// AccessOthersBot 他人のBotのアクセス権限
	AccessOthersBot = rbac.Permission("access_others_bot")

	// BotActionJoinChannel BOTアクション実行権限：チャンネル参加
	BotActionJoinChannel = rbac.Permission("bot_action_join_channel")
	// BotActionLeaveChannel BOTアクション実行権限：チャンネル退出
	BotActionLeaveChannel = rbac.Permission("bot_action_leave_channel")
)
