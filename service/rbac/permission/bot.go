package permission

const (
	// GetWebhook Webhook情報取得権限
	GetWebhook = Permission("get_webhook")
	// CreateWebhook Webhook作成権限
	CreateWebhook = Permission("create_webhook")
	// EditWebhook Webhook編集権限
	EditWebhook = Permission("edit_webhook")
	// DeleteWebhook Webhook削除権限
	DeleteWebhook = Permission("delete_webhook")
	// AccessOthersWebhook 他人のWebhookのアクセス権限
	AccessOthersWebhook = Permission("access_others_webhook")

	// GetBot Bot情報取得権限
	GetBot = Permission("get_bot")
	// CreateBot Bot作成権限
	CreateBot = Permission("create_bot")
	// EditBot Bot編集権限
	EditBot = Permission("edit_bot")
	// DeleteBot Bot削除権限
	DeleteBot = Permission("delete_bot")
	// AccessOthersBot 他人のBotのアクセス権限
	AccessOthersBot = Permission("access_others_bot")

	// BotActionJoinChannel BOTアクション実行権限：チャンネル参加
	BotActionJoinChannel = Permission("bot_action_join_channel")
	// BotActionLeaveChannel BOTアクション実行権限：チャンネル退出
	BotActionLeaveChannel = Permission("bot_action_leave_channel")
)
