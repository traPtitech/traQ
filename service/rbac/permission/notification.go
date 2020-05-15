package permission

const (
	// GetChannelSubscription チャンネル購読状況取得権限
	GetChannelSubscription = Permission("get_channel_subscription")
	// EditChannelSubscription チャンネル購読変更権限
	EditChannelSubscription = Permission("edit_channel_subscription")
	// ConnectNotificationStream 通知ストリームへの接続権限
	ConnectNotificationStream = Permission("connect_notification_stream")
	// RegisterFCMDevice FCMデバイスの登録権限
	RegisterFCMDevice = Permission("register_fcm_device")
)
