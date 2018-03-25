package permission

import "github.com/mikespook/gorbac"

var (
	// GetWebhook : Webhook情報取得権限
	GetWebhook = gorbac.NewStdPermission("get_webhook")
	// CreateWebhook : Webhook作成権限
	CreateWebhook = gorbac.NewStdPermission("create_webhook")
	// EditWebhook : Webhook編集権限
	EditWebhook = gorbac.NewStdPermission("edit_webhook")
	// DeleteWebhook : Webhook削除権限
	DeleteWebhook = gorbac.NewStdPermission("delete_webhook")
)
