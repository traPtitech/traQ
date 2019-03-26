package permission

import "github.com/mikespook/gorbac"

var (
	// GetWebhook Webhook情報取得権限
	GetWebhook = gorbac.NewStdPermission("get_webhook")
	// CreateWebhook Webhook作成権限
	CreateWebhook = gorbac.NewStdPermission("create_webhook")
	// EditWebhook Webhook編集権限
	EditWebhook = gorbac.NewStdPermission("edit_webhook")
	// DeleteWebhook Webhook削除権限
	DeleteWebhook = gorbac.NewStdPermission("delete_webhook")

	// GetBot Bot情報取得権限
	GetBot = gorbac.NewStdPermission("get_bot")
	// CreateBot Bot作成権限
	CreateBot = gorbac.NewStdPermission("create_bot")
	// EditBot Bot編集権限
	EditBot = gorbac.NewStdPermission("edit_bot")
	// DeleteBot Bot削除権限
	DeleteBot = gorbac.NewStdPermission("delete_bot")
	// ReissueBotToken Botトークン再発行権限
	ReissueBotToken = gorbac.NewStdPermission("reissue_bot_token")
	// InstallBot Botインストール権限
	InstallBot = gorbac.NewStdPermission("install_bot")
	// UninstallBot Botアンインストール権限
	UninstallBot = gorbac.NewStdPermission("uninstall_bot")
)
