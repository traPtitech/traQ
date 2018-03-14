package permission

import "github.com/mikespook/gorbac"

var (
	// GetNotificationStatus : チャンネルの通知状況取得権限
	GetNotificationStatus = gorbac.NewStdPermission("get_notification_status")
	// ChangeNotificationStatus : チャンネルの通知状況変更権限
	ChangeNotificationStatus = gorbac.NewStdPermission("change_notification_status")
	// ConnectNotificationStream : 通知ストリームへの接続権限
	ConnectNotificationStream = gorbac.NewStdPermission("connect_notification_stream")
	// RegisterDevice : 通知デバイスの登録権限
	RegisterDevice = gorbac.NewStdPermission("register_device")
)
