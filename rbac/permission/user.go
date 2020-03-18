package permission

import (
	"github.com/traPtitech/traQ/rbac"
)

const (
	// GetUser ユーザー情報取得権限
	GetUser = rbac.Permission("get_user")
	// RegisterUser 新規ユーザー登録権限
	RegisterUser = rbac.Permission("register_user")
	// GetMe 自ユーザー情報取得権限
	GetMe = rbac.Permission("get_me")
	// EditMe 自ユーザー情報変更権限
	EditMe = rbac.Permission("edit_me")
	// ChangeMyIcon 自ユーザーアイコン変更権限
	ChangeMyIcon = rbac.Permission("change_my_icon")
	// ChangeMyPassword 自ユーザーパスワード変更権限
	ChangeMyPassword = rbac.Permission("change_my_password")
	// EditOtherUsers 他ユーザー情報変更権限
	EditOtherUsers = rbac.Permission("edit_other_users")
	// GetUserQRCode ユーザーQRコード取得権限
	GetUserQRCode = rbac.Permission("get_user_qr_code")
	// GetUserTag ユーザータグ取得権限
	GetUserTag = rbac.Permission("get_user_tag")
	// EditUserTag ユーザータグ編集権限
	EditUserTag = rbac.Permission("edit_user_tag")
	// GetUserGroup ユーザーグループ取得権限
	GetUserGroup = rbac.Permission("get_user_group")
	// CreateUserGroup ユーザーグループ作成権限
	CreateUserGroup = rbac.Permission("create_user_group")
	// CreateSpecialUserGroup 特殊ユーザーグループ作成権限
	CreateSpecialUserGroup = rbac.Permission("create_special_user_group")
	// EditUserGroup ユーザーグループ編集権限
	EditUserGroup = rbac.Permission("edit_user_group")
	// DeleteUserGroup ユーザーグループ削除権限
	DeleteUserGroup = rbac.Permission("delete_user_group")
	// GetHeartbeat ハートビート取得権限
	GetHeartbeat = rbac.Permission("get_heartbeat")
	// PostHeartbeat ハートビート送信権限
	PostHeartbeat = rbac.Permission("post_heartbeat")
	// WebRTC WebRTC利用権限
	WebRTC = rbac.Permission("web_rtc")
	// GetMySessions セッションリスト取得権限
	GetMySessions = rbac.Permission("get_my_sessions")
	// DeleteMySessions セッション削除権限
	DeleteMySessions = rbac.Permission("delete_my_sessions")
	// GetUnread 未読メッセージ一覧の取得権限
	GetUnread = rbac.Permission("get_unread")
	// DeleteUnread メッセージ既読化権限
	DeleteUnread = rbac.Permission("delete_unread")

	// GetClipFolder クリップフォルダ取得権限
	GetClipFolder = rbac.Permission("get_clip_folder")
	// CreateClipFolder クリップフォルダ作成権限
	CreateClipFolder = rbac.Permission("create_clip_folder")
	// EditClipFolder クリップフォルダ編集権限
	EditClipFolder = rbac.Permission("edit_clip_folder")
	// DeleteClipFolder クリップフォルダ削除権限
	DeleteClipFolder = rbac.Permission("delete_clip_folder")
)
