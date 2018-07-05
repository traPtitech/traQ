package permission

import "github.com/mikespook/gorbac"

var (
	// CreateChannel チャンネル作成権限
	CreateChannel = gorbac.NewStdPermission("create_channel")
	// GetChannel チャンネル情報取得権限
	GetChannel = gorbac.NewStdPermission("get_channel")
	// EditChannel チャンネル情報変更権限
	EditChannel = gorbac.NewStdPermission("edit_channel")
	// DeleteChannel チャンネル削除権限
	DeleteChannel = gorbac.NewStdPermission("delete_channel")
	// ChangeParentChannel 親チャンネル変更権限
	ChangeParentChannel = gorbac.NewStdPermission("change_parent_channel")
)
