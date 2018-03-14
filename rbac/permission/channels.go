package permission

import "github.com/mikespook/gorbac"

var (
	// GetChannels : チャンネルリスト取得権限
	GetChannels = gorbac.NewStdPermission("get_channels")
	// CreateChannels : チャンネル作成権限
	CreateChannels = gorbac.NewStdPermission("create_channels")
	// GetChannel : チャンネル情報取得権限
	GetChannel = gorbac.NewStdPermission("get_channel")
	// PatchChannel : チャンネル情報変更権限
	PatchChannel = gorbac.NewStdPermission("patch_channel")
	// DeleteChannel : チャンネル削除権限
	DeleteChannel = gorbac.NewStdPermission("delete_channel")
)
