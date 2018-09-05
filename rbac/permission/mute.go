package permission

import "github.com/mikespook/gorbac"

var (
	// GetMutedChannels ミュートチャンネル一覧取得権限
	GetMutedChannels = gorbac.NewStdPermission("get_muted_channels")
	// MuteChannel チャンネルミュート権限
	MuteChannel = gorbac.NewStdPermission("mute_channel")
	// UnmuteChannel チャンネルアンミュート権限
	UnmuteChannel = gorbac.NewStdPermission("unmute_channel")
)
