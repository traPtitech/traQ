package permission

import "github.com/mikespook/gorbac"

// 全パーミッションのリスト。パーミッションを新たに定義した場合はここに必ず追加すること
var list = map[string]gorbac.Permission{
	GetChannels.ID():    GetChannels,
	CreateChannels.ID(): CreateChannels,
	GetChannel.ID():     GetChannel,
	PatchChannel.ID():   PatchChannel,
	DeleteChannel.ID():  DeleteChannel,
}

// GetPermission : パーミッション名からgorbac.Permissionを取得します
func GetPermission(name string) gorbac.Permission {
	return list[name]
}
