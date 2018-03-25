package permission

import "github.com/mikespook/gorbac"

var (
	// GetPin : ピン留め取得権限
	GetPin = gorbac.NewStdPermission("get_pin")
	// CreatePin : ピン留め作成権限
	CreatePin = gorbac.NewStdPermission("create_pin")
	// DeletePin : ピン留め削除権限
	DeletePin = gorbac.NewStdPermission("delete_pin")
)
