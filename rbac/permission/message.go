package permission

import "github.com/mikespook/gorbac"

var (
	// GetMessage メッセージ取得権限
	GetMessage = gorbac.NewStdPermission("get_message")
	// PostMessage メッセージ投稿権限
	PostMessage = gorbac.NewStdPermission("post_message")
	// EditMessage メッセージ編集権限
	EditMessage = gorbac.NewStdPermission("edit_message")
	// DeleteMessage メッセージ削除権限
	DeleteMessage = gorbac.NewStdPermission("delete_message")
	// ReportMessage メッセージ通報権限
	ReportMessage = gorbac.NewStdPermission("report_message")
	// GetMessageReports メッセージ通報取得権限
	GetMessageReports = gorbac.NewStdPermission("get_message_reports")
)
