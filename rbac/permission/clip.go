package permission

import (
	"github.com/traPtitech/traQ/rbac"
)

const (
	// GetClip クリップ取得権限
	GetClip = rbac.Permission("get_clip")
	// CreateClip クリップ作成権限
	CreateClip = rbac.Permission("create_clip")
	// DeleteClip クリップ削除権限
	DeleteClip = rbac.Permission("delete_clip")
	// GetClipFolder クリップフォルダ取得権限
	GetClipFolder = rbac.Permission("get_clip_folder")
	// CreateClipFolder クリップフォルダ作成権限
	CreateClipFolder = rbac.Permission("create_clip_folder")
	// PatchClipFolder クリップフォルダ修正権限
	PatchClipFolder = rbac.Permission("patch_clip_folder")
	// DeleteClipFolder クリップフォルダ削除権限
	DeleteClipFolder = rbac.Permission("delete_clip_folder")
)
