package permission

const (
	// GetUser ユーザー情報取得権限
	GetUser = Permission("get_user")
	// RegisterUser 新規ユーザー登録権限
	RegisterUser = Permission("register_user")
	// GetMe 自ユーザー情報取得権限
	GetMe = Permission("get_me")
	// GetOIDCUserInfo 自ユーザー情報取得権限 (OIDC専用)
	GetOIDCUserInfo = Permission("get_oidc_userinfo")
	// EditMe 自ユーザー情報変更権限
	EditMe = Permission("edit_me")
	// ChangeMyIcon 自ユーザーアイコン変更権限
	ChangeMyIcon = Permission("change_my_icon")
	// ChangeMyPassword 自ユーザーパスワード変更権限
	ChangeMyPassword = Permission("change_my_password")
	// EditOtherUsers 他ユーザー情報変更権限
	EditOtherUsers = Permission("edit_other_users")
	// GetUserQRCode ユーザーQRコード取得権限
	GetUserQRCode = Permission("get_user_qr_code")
	// GetUserTag ユーザータグ取得権限
	GetUserTag = Permission("get_user_tag")
	// EditUserTag ユーザータグ編集権限
	EditUserTag = Permission("edit_user_tag")
	// GetUserGroup ユーザーグループ取得権限
	GetUserGroup = Permission("get_user_group")
	// CreateUserGroup ユーザーグループ作成権限
	CreateUserGroup = Permission("create_user_group")
	// CreateSpecialUserGroup 特殊ユーザーグループ作成権限
	CreateSpecialUserGroup = Permission("create_special_user_group")
	// EditUserGroup ユーザーグループ編集権限
	EditUserGroup = Permission("edit_user_group")
	// DeleteUserGroup ユーザーグループ削除権限
	DeleteUserGroup = Permission("delete_user_group")
	// AllUserGroupsAdmin すべてのユーザーグループの編集/削除権限
	AllUserGroupsAdmin = Permission("edit_others_user_group")
	// WebRTC WebRTC利用権限
	WebRTC = Permission("web_rtc")
	// GetMySessions セッションリスト取得権限
	GetMySessions = Permission("get_my_sessions")
	// DeleteMySessions セッション削除権限
	DeleteMySessions = Permission("delete_my_sessions")
	// GetMyExternalAccount 外部ログインアカウント情報取得権限
	GetMyExternalAccount = Permission("get_my_external_account")
	// EditMyExternalAccount 外部ログインアカウント情報編集権限
	EditMyExternalAccount = Permission("edit_my_external_account")
	// GetUnread 未読メッセージ一覧の取得権限
	GetUnread = Permission("get_unread")
	// DeleteUnread メッセージ既読化権限
	DeleteUnread = Permission("delete_unread")

	// GetClipFolder クリップフォルダ取得権限
	GetClipFolder = Permission("get_clip_folder")
	// CreateClipFolder クリップフォルダ作成権限
	CreateClipFolder = Permission("create_clip_folder")
	// EditClipFolder クリップフォルダ編集権限
	EditClipFolder = Permission("edit_clip_folder")
	// DeleteClipFolder クリップフォルダ削除権限
	DeleteClipFolder = Permission("delete_clip_folder")
)
