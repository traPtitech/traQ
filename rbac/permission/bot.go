package permission

import (
	"github.com/traPtitech/traQ/rbac"
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
	// InstallBot Botインストール権限
	InstallBot = rbac.Permission("install_bot")
	// UninstallBot Botアンインストール権限
	UninstallBot = rbac.Permission("uninstall_bot")
	// AccessOthersBot 他人のBotのアクセス権限
	AccessOthersBot = rbac.Permission("access_others_bot")
)
