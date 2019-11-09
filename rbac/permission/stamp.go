package permission

import (
	"github.com/traPtitech/traQ/rbac"
)

const (
	// GetStamp スタンプ情報取得権限
	GetStamp = rbac.Permission("get_stamp")
	// CreateStamp スタンプ作成権限
	CreateStamp = rbac.Permission("create_stamp")
	// EditStamp 自スタンプ画像変更権限
	EditStamp = rbac.Permission("edit_stamp")
	// EditStampName 自スタンプ名変更権限
	EditStampName = rbac.Permission("edit_stamp_name")
	// EditStampCreatedByOthers 他ユーザー作成のスタンプの変更権限
	EditStampCreatedByOthers = rbac.Permission("edit_stamp_created_by_others")
	// DeleteStamp スタンプ削除権限
	DeleteStamp = rbac.Permission("delete_stamp")
	// AddMessageStamp メッセージスタンプ追加権限
	AddMessageStamp = rbac.Permission("add_message_stamp")
	// RemoveMessageStamp メッセージスタンプ削除権限
	RemoveMessageStamp = rbac.Permission("remove_message_stamp")
	// GetMyStampHistory 自分のスタンプ履歴取得権限
	GetMyStampHistory = rbac.Permission("get_my_stamp_history")
)
