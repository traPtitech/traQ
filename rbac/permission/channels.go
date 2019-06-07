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
	// GetTopic : チャンネルトピック取得権限
	GetTopic = rbac.Permission("get_topic")
	// EditTopic : チャンネルトピック変更権限
	EditTopic = rbac.Permission("edit_topic")
	// GetChannelVisibility : チャンネルの可視状態の取得権限
	GetChannelVisibility = rbac.Permission("get_channel_visibility")
	// ChangeChannelVisibility : チャンネルの可視状態の変更権限
	ChangeChannelVisibility = rbac.Permission("change_channel_visibility")
)
