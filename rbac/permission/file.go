package permission

import (
	"github.com/traPtitech/traQ/rbac"
)

const (
	// UploadFile : ファイルアップロード権限
	UploadFile = rbac.Permission("upload_file")
	// DownloadFile : ファイルダウンロード権限
	DownloadFile = rbac.Permission("download_file")
	// DeleteFile : ファイル削除権限
	DeleteFile = rbac.Permission("delete_file")
)
