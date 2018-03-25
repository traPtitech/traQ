package permission

import "github.com/mikespook/gorbac"

var (
	// GetStar : スター取得権限
	GetStar = gorbac.NewStdPermission("get_star")
	// CreateStar : スター作成権限
	CreateStar = gorbac.NewStdPermission("create_star")
	// DeleteStar : スター削除権限
	DeleteStar = gorbac.NewStdPermission("delete_star")
)
