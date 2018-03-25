package permission

import "github.com/mikespook/gorbac"

var (
	// GetChannelVisibility : チャンネルの可視状態の取得権限
	GetChannelVisibility = gorbac.NewStdPermission("get_channel_visibility")
	// ChangeChannelVisibility : チャンネルの可視状態の変更権限
	ChangeChannelVisibility = gorbac.NewStdPermission("change_channel_visibility")
)
