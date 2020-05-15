package permission

const (
	// UploadFile ファイルアップロード権限
	UploadFile = Permission("upload_file")
	// DownloadFile ファイルダウンロード権限
	DownloadFile = Permission("download_file")
	// DeleteFile ファイル削除権限
	DeleteFile = Permission("delete_file")
)
