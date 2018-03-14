package permission

import "github.com/mikespook/gorbac"

var list = map[string]gorbac.Permission{}

// GetPermission : パーミッション名からgorbac.Permissionを取得します
func GetPermission(name string) gorbac.Permission {
	return list[name]
}
