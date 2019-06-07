package permission

import "github.com/traPtitech/traQ/rbac"

const (
	// GetNotificationStatus : チャンネルの通知状況取得権限
	GetNotificationStatus = rbac.Permission("get_notification_status")
	// ChangeNotificationStatus : チャンネルの通知状況変更権限
	ChangeNotificationStatus = rbac.Permission("change_notification_status")
	// ConnectNotificationStream : 通知ストリームへの接続権限
	ConnectNotificationStream = rbac.Permission("connect_notification_stream")
	// RegisterDevice : 通知デバイスの登録権限
	RegisterDevice = rbac.Permission("register_device")
)
