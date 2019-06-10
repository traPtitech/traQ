package permission

import (
	"github.com/traPtitech/traQ/rbac"
)

const (
	// CreateChannel チャンネル作成権限
	CreateChannel = rbac.Permission("create_channel")
	// GetChannel チャンネル情報取得権限
	GetChannel = rbac.Permission("get_channel")
	// EditChannel チャンネル情報変更権限
	EditChannel = rbac.Permission("edit_channel")
	// DeleteChannel チャンネル削除権限
	DeleteChannel = rbac.Permission("delete_channel")
	// ChangeParentChannel 親チャンネル変更権限
	ChangeParentChannel = rbac.Permission("change_parent_channel")
	// EditChannelTopic チャンネルトピック変更権限
	EditChannelTopic = rbac.Permission("edit_channel_topic")
	// GetChannelStar チャンネルスター取得権限
	GetChannelStar = rbac.Permission("get_channel_star")
	// EditChannelStar チャンネルスター編集権限
	EditChannelStar = rbac.Permission("edit_channel_star")
	// GetChannelMute チャンネルミュート取得権限
	GetChannelMute = rbac.Permission("get_channel_mute")
	// EditChannelMute チャンネルミュート編集権限
	EditChannelMute = rbac.Permission("edit_channel_mute")
)
