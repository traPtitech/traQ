package migration

import (
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/router/sessions"
	"gopkg.in/gormigrate.v1"
)

// Migrations 全てのデータベースマイグレーション
//
// 新たなマイグレーションを行う場合は、この配列の末尾に必ず追加すること
func Migrations() []*gormigrate.Migration {
	return []*gormigrate.Migration{
		v1(),  // インデックスidx_messages_deleted_atの削除とidx_messages_channel_id_deleted_at_created_atの追加
		v2(),  // RBAC周りのリフォーム
		v3(),  // チャンネルイベント履歴
		v4(),  // Webhook, Bot外部キー
		v5(),  // Mute, 旧Clip削除
		v6(),  // v6 ユーザーグループ拡張
		v7(),  // ファイルメタ拡張
		v8(),  // チャンネル購読拡張
		v9(),  // ユーザーテーブル拡張
		v10(), // パーミッション周りの調整
		v11(), // クリップ機能の追加
		v12(), // カスタムスタンプパレットの追加
		v13(), // パーミッション調整・インデックス付与
		v14(), // パーミッション不足修正
	}
}

// AllTables 最新のスキーマの全テーブルモデル
//
// 最新のスキーマの全テーブルのモデル構造体を記述すること
func AllTables() []interface{} {
	return []interface{}{
		&model.ChannelEvent{},
		&model.RolePermission{},
		&model.RoleInheritance{},
		&model.UserRole{},
		&model.DMChannelMapping{},
		&model.ChannelLatestMessage{},
		&model.BotEventLog{},
		&model.BotJoinChannel{},
		&model.Bot{},
		&model.OAuth2Client{},
		&model.OAuth2Authorize{},
		&model.OAuth2Token{},
		&model.MessageReport{},
		&model.WebhookBot{},
		&model.MessageStamp{},
		&model.Stamp{},
		&model.UsersTag{},
		&model.Unread{},
		&model.Star{},
		&model.Device{},
		&model.Pin{},
		&model.FileACLEntry{},
		&model.File{},
		&model.UsersPrivateChannel{},
		&model.UserSubscribeChannel{},
		&model.Tag{},
		&model.ArchivedMessage{},
		&model.ClipFolderMessage{},
		&model.Message{},
		&model.Channel{},
		&model.StampPalette{},
		&model.UserGroupAdmin{},
		&model.UserGroupMember{},
		&model.UserGroup{},
		&model.UserProfile{},
		&model.ClipFolder{},
		&model.User{},
		&sessions.SessionRecord{},
	}
}

// AllForeignKeys 最新のスキーマの全外部キー制約
//
// 最新のスキーマの全外部キー制約を記述すること
func AllForeignKeys() [][5]string {
	return [][5]string{
		// Table, Key, Reference, OnDelete, OnUpdate
		{"user_role_inheritances", "role", "user_roles(name)", "CASCADE", "CASCADE"},
		{"user_role_inheritances", "sub_role", "user_roles(name)", "CASCADE", "CASCADE"},
		{"user_role_permissions", "role", "user_roles(name)", "CASCADE", "CASCADE"},
		{"users_private_channels", "user_id", "users(id)", "CASCADE", "CASCADE"},
		{"users_private_channels", "channel_id", "channels(id)", "CASCADE", "CASCADE"},
		{"dm_channel_mappings", "channel_id", "channels(id)", "CASCADE", "CASCADE"},
		{"dm_channel_mappings", "user1", "users(id)", "CASCADE", "CASCADE"},
		{"dm_channel_mappings", "user2", "users(id)", "CASCADE", "CASCADE"},
		{"messages", "user_id", "users(id)", "CASCADE", "CASCADE"},
		{"messages", "channel_id", "channels(id)", "CASCADE", "CASCADE"},
		{"users_tags", "user_id", "users(id)", "CASCADE", "CASCADE"},
		{"users_tags", "tag_id", "tags(id)", "CASCADE", "CASCADE"},
		{"unreads", "user_id", "users(id)", "CASCADE", "CASCADE"},
		{"unreads", "message_id", "messages(id)", "CASCADE", "CASCADE"},
		{"devices", "user_id", "users(id)", "CASCADE", "CASCADE"},
		{"stars", "user_id", "users(id)", "CASCADE", "CASCADE"},
		{"stars", "channel_id", "channels(id)", "CASCADE", "CASCADE"},
		{"users_subscribe_channels", "user_id", "users(id)", "CASCADE", "CASCADE"},
		{"users_subscribe_channels", "channel_id", "channels(id)", "CASCADE", "CASCADE"},
		{"pins", "user_id", "users(id)", "CASCADE", "CASCADE"},
		{"pins", "message_id", "messages(id)", "CASCADE", "CASCADE"},
		{"messages_stamps", "message_id", "messages(id)", "CASCADE", "CASCADE"},
		{"messages_stamps", "stamp_id", "stamps(id)", "CASCADE", "CASCADE"},
		{"messages_stamps", "user_id", "users(id)", "CASCADE", "CASCADE"},
		{"stamps", "file_id", "files(id)", "NO ACTION", "CASCADE"},
		{"webhook_bots", "bot_user_id", "users(id)", "CASCADE", "CASCADE"},
		{"webhook_bots", "creator_id", "users(id)", "CASCADE", "CASCADE"},
		{"webhook_bots", "channel_id", "channels(id)", "CASCADE", "CASCADE"},
		{"bots", "creator_id", "users(id)", "CASCADE", "CASCADE"},
		{"bots", "bot_user_id", "users(id)", "CASCADE", "CASCADE"},
		{"channel_events", "channel_id", "channels(id)", "CASCADE", "CASCADE"},
		{"files", "channel_id", "channels(id)", "SET NULL", "CASCADE"},
		{"files", "creator_id", "users(id)", "RESTRICT", "CASCADE"},
		{"files_acl", "file_id", "files(id)", "CASCADE", "CASCADE"},
		{"user_profiles", "user_id", "users(id)", "CASCADE", "CASCADE"},
		{"clip_folders", "owner_id", "users(id)", "CASCADE", "CASCADE"},
		{"clip_folder_messages", "folder_id", "clip_folders(id)", "CASCADE", "CASCADE"},
		{"clip_folder_messages", "message_id", "messages(id)", "CASCADE", "CASCADE"},
		{"stamp_palettes", "creator_id", "users(id)", "CASCADE", "CASCADE"},
	}
}

// AllCompositeIndexes 最新のスキーマの全複合インデックス
//
// 最新のスキーマの全複合インデックスを記述すること。
func AllCompositeIndexes() [][]string {
	return [][]string{
		// Name,  Table, Columns...
		{"idx_messages_channel_id_deleted_at_created_at", "messages", "channel_id", "deleted_at", "created_at"},
		{"idx_channel_events_channel_id_date_time", "channel_events", "channel_id", "date_time"},
		{"idx_channel_events_channel_id_event_type_date_time", "channel_events", "channel_id", "event_type", "date_time"},
		{"idx_files_channel_id_created_at", "files", "channel_id", "created_at"},
		{"idx_files_creator_id_created_at", "files", "creator_id", "created_at"},
		{"idx_messages_stamps_user_id_stamp_id_updated_at", "messages_stamps", "user_id", "stamp_id", "updated_at"},
	}
}
