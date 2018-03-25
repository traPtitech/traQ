package permission

import "github.com/mikespook/gorbac"

var (
	// UploadFile : ファイルアップロード権限
	UploadFile = gorbac.NewStdPermission("upload_file")
	// DownloadFile : ファイルダウンロード権限
	DownloadFile = gorbac.NewStdPermission("download_file")
	// DeleteFile : ファイル削除権限
	DeleteFile = gorbac.NewStdPermission("delete_file")
)
