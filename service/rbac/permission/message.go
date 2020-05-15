package permission

const (
	// GetMessage メッセージ取得権限
	GetMessage = Permission("get_message")
	// PostMessage メッセージ投稿権限
	PostMessage = Permission("post_message")
	// EditMessage メッセージ編集権限
	EditMessage = Permission("edit_message")
	// DeleteMessage メッセージ削除権限
	DeleteMessage = Permission("delete_message")
	// ReportMessage メッセージ通報権限
	ReportMessage = Permission("report_message")
	// GetMessageReports メッセージ通報取得権限
	GetMessageReports = Permission("get_message_reports")
	// CreateMessagePin ピン留め作成権限
	CreateMessagePin = Permission("create_message_pin")
	// DeleteMessagePin ピン留め削除権限
	DeleteMessagePin = Permission("delete_message_pin")
)
