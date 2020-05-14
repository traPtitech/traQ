package permission

import (
	"github.com/traPtitech/traQ/service/rbac"
)

const (
	// GetMessage メッセージ取得権限
	GetMessage = rbac.Permission("get_message")
	// PostMessage メッセージ投稿権限
	PostMessage = rbac.Permission("post_message")
	// EditMessage メッセージ編集権限
	EditMessage = rbac.Permission("edit_message")
	// DeleteMessage メッセージ削除権限
	DeleteMessage = rbac.Permission("delete_message")
	// ReportMessage メッセージ通報権限
	ReportMessage = rbac.Permission("report_message")
	// GetMessageReports メッセージ通報取得権限
	GetMessageReports = rbac.Permission("get_message_reports")
	// CreateMessagePin ピン留め作成権限
	CreateMessagePin = rbac.Permission("create_message_pin")
	// DeleteMessagePin ピン留め削除権限
	DeleteMessagePin = rbac.Permission("delete_message_pin")
)
