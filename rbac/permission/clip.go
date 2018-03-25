package permission

import "github.com/mikespook/gorbac"

var (
	// GetClip : クリップ取得権限
	GetClip = gorbac.NewStdPermission("get_clip")
	// CreateClip : クリップ作成権限
	CreateClip = gorbac.NewStdPermission("create_clip")
	// DeleteClip : クリップ削除権限
	DeleteClip = gorbac.NewStdPermission("delete_clip")
)
