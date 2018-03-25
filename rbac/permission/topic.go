package permission

import "github.com/mikespook/gorbac"

var (
	// GetTopic : チャンネルトピック取得権限
	GetTopic = gorbac.NewStdPermission("get_topic")
	// EditTopic : チャンネルトピック変更権限
	EditTopic = gorbac.NewStdPermission("edit_topic")
)
