package permission

const (
	// CreateChannel チャンネル作成権限
	CreateChannel = Permission("create_channel")
	// GetChannel チャンネル情報取得権限
	GetChannel = Permission("get_channel")
	// EditChannel チャンネル情報変更権限
	EditChannel = Permission("edit_channel")
	// DeleteChannel チャンネル削除権限
	DeleteChannel = Permission("delete_channel")
	// ChangeParentChannel 親チャンネル変更権限
	ChangeParentChannel = Permission("change_parent_channel")
	// EditChannelTopic チャンネルトピック変更権限
	EditChannelTopic = Permission("edit_channel_topic")
	// GetChannelStar チャンネルスター取得権限
	GetChannelStar = Permission("get_channel_star")
	// EditChannelStar チャンネルスター編集権限
	EditChannelStar = Permission("edit_channel_star")
)
