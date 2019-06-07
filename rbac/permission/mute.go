package permission

import (
	"github.com/traPtitech/traQ/rbac"
)

const (
	// GetMutedChannels ミュートチャンネル一覧取得権限
	GetMutedChannels = rbac.Permission("get_muted_channels")
	// MuteChannel チャンネルミュート権限
	MuteChannel = rbac.Permission("mute_channel")
	// UnmuteChannel チャンネルアンミュート権限
	UnmuteChannel = rbac.Permission("unmute_channel")
)
