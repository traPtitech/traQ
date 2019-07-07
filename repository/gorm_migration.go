package repository

import (
	"github.com/jinzhu/gorm"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/rbac/role"
	"github.com/traPtitech/traQ/repository/migration"
	"gopkg.in/gormigrate.v1"
)

func (repo *GormRepository) migration() error {
	m := gormigrate.New(repo.db, &gormigrate.Options{
		TableName:      "migrations",
		IDColumnName:   "id",
		IDColumnSize:   190,
		UseTransaction: false,
	}, migrations)
	m.InitSchema(func(db *gorm.DB) error {
		// 初回のみに呼ばれる
		// 全ての最新のデータベース定義を書く事

		// テーブル
		if err := db.AutoMigrate(allTables...).Error; err != nil {
			return err
		}

		// 外部キー制約
		foreignKeys := [][5]string{
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
			{"mutes", "user_id", "users(id)", "CASCADE", "CASCADE"},
			{"mutes", "channel_id", "channels(id)", "CASCADE", "CASCADE"},
			{"users_subscribe_channels", "user_id", "users(id)", "CASCADE", "CASCADE"},
			{"users_subscribe_channels", "channel_id", "channels(id)", "CASCADE", "CASCADE"},
			{"clips", "folder_id", "clip_folders(id)", "CASCADE", "CASCADE"},
			{"clips", "message_id", "messages(id)", "CASCADE", "CASCADE"},
			{"clips", "user_id", "users(id)", "CASCADE", "CASCADE"},
			{"clip_folders", "user_id", "users(id)", "CASCADE", "CASCADE"},
			{"pins", "user_id", "users(id)", "CASCADE", "CASCADE"},
			{"pins", "message_id", "messages(id)", "CASCADE", "CASCADE"},
			{"messages_stamps", "message_id", "messages(id)", "CASCADE", "CASCADE"},
			{"messages_stamps", "stamp_id", "stamps(id)", "CASCADE", "CASCADE"},
			{"messages_stamps", "user_id", "users(id)", "CASCADE", "CASCADE"},
			{"stamps", "file_id", "files(id)", "NO ACTION", "CASCADE"},
			{"webhook_bots", "bot_user_id", "users(id)", "CASCADE", "CASCADE"},
			{"channel_events", "channel_id", "channels(id)", "CASCADE", "CASCADE"},
		}
		for _, c := range foreignKeys {
			if err := db.Table(c[0]).AddForeignKey(c[1], c[2], c[3], c[4]).Error; err != nil {
				return err
			}
		}

		// 複合インデックス
		indexes := [][]string{
			// Name,  Table, Columns...
			{"idx_messages_channel_id_deleted_at_created_at", "messages", "channel_id", "deleted_at", "created_at"},
			{"idx_channel_events_channel_id_date_time", "channel_events", "channel_id", "date_time"},
			{"idx_channel_events_channel_id_event_type_date_time", "channel_events", "channel_id", "event_type", "date_time"},
		}
		for _, v := range indexes {
			if err := db.Table(v[1]).AddIndex(v[0], v[2:]...).Error; err != nil {
				return err
			}
		}

		// 初期ユーザーロール投入
		for _, v := range role.SystemRoles() {
			if err := db.Create(v).Error; err != nil {
				return err
			}

			for _, v := range v.Permissions {
				if err := db.Create(v).Error; err != nil {
					return err
				}
			}
		}

		return nil
	})
	return m.Migrate()
}

// 全テーブル
var allTables = []interface{}{
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
	&model.Mute{},
	&model.MessageReport{},
	&model.WebhookBot{},
	&model.MessageStamp{},
	&model.Stamp{},
	&model.Clip{},
	&model.ClipFolder{},
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
	&model.Message{},
	&model.Channel{},
	&model.UserGroupMember{},
	&model.UserGroup{},
	&model.User{},
}

// データベースマイグレーション
var migrations = []*gormigrate.Migration{
	migration.V1, // インデックスidx_messages_deleted_atの削除とidx_messages_channel_id_deleted_at_created_atの追加
	migration.V2, // RBAC周りのリフォーム
	migration.V3, // チャンネルイベント履歴
}
