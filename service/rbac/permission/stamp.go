package permission

const (
	// GetStamp スタンプ情報取得権限
	GetStamp = Permission("get_stamp")
	// CreateStamp スタンプ作成権限
	CreateStamp = Permission("create_stamp")
	// EditStamp 自スタンプ画像変更権限
	EditStamp = Permission("edit_stamp")
	// EditStampCreatedByOthers 他ユーザー作成のスタンプの変更権限
	EditStampCreatedByOthers = Permission("edit_stamp_created_by_others")
	// DeleteStamp スタンプ削除権限
	DeleteStamp = Permission("delete_stamp")
	// DeleteMyStamp 自分のスタンプ削除権限
	DeleteMyStamp = Permission("delete_my_stamp")
	// AddMessageStamp メッセージスタンプ追加権限
	AddMessageStamp = Permission("add_message_stamp")
	// RemoveMessageStamp メッセージスタンプ削除権限
	RemoveMessageStamp = Permission("remove_message_stamp")
	// GetMyStampHistory 自分のスタンプ履歴取得権限
	GetMyStampHistory = Permission("get_my_stamp_history")

	// GetStampPalette スタンプパレット取得権限
	GetStampPalette = Permission("get_stamp_palette")
	// CreateStampPalette スタンプパレット作成権限
	CreateStampPalette = Permission("create_stamp_palette")
	// EditStampPalette スタンプパレット編集権限
	EditStampPalette = Permission("edit_stamp_palette")
	// DeleteStampPalette スタンプパレット削除権限
	DeleteStampPalette = Permission("delete_stamp_palette")
)
