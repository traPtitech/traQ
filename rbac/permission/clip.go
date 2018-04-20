package permission

import "github.com/mikespook/gorbac"

var (
	// GetClip クリップ取得権限
	GetClip = gorbac.NewStdPermission("get_clip")
	// CreateClip クリップ作成権限
	CreateClip = gorbac.NewStdPermission("create_clip")
	// DeleteClip クリップ削除権限
	DeleteClip = gorbac.NewStdPermission("delete_clip")
	// GetClipFolder クリップフォルダ取得権限
	GetClipFolder = gorbac.NewStdPermission("get_clip_folder")
	// CreateClipFolder クリップフォルダ作成権限
	CreateClipFolder = gorbac.NewStdPermission("create_clip_folder")
	// PatchClipFolder クリップフォルダ修正権限
	PatchClipFolder = gorbac.NewStdPermission("patch_clip_folder")
	// DeleteClipFolder クリップフォルダ削除権限
	DeleteClipFolder = gorbac.NewStdPermission("delete_clip_folder")
)
