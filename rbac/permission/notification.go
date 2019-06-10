package permission

import "github.com/traPtitech/traQ/rbac"

const (
	// GetChannelSubscription チャンネル購読状況取得権限
	GetChannelSubscription = rbac.Permission("get_channel_subscription")
	// EditChannelSubscription チャンネル購読変更権限
	EditChannelSubscription = rbac.Permission("edit_channel_subscription")
	// ConnectNotificationStream 通知ストリームへの接続権限
	ConnectNotificationStream = rbac.Permission("connect_notification_stream")
	// RegisterFCMDevice FCMデバイスの登録権限
	RegisterFCMDevice = rbac.Permission("register_fcm_device")
)
