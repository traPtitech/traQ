package migration

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// v1 インデックスidx_messages_deleted_atの削除とidx_messages_channel_id_deleted_at_created_atの追加
func v1() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "1",
		Migrate: func(db *gorm.DB) error {
			if err := db.Migrator().DropIndex("messages", "idx_messages_deleted_at"); err != nil {
				return err
			}
			return db.Exec("ALTER TABLE `messages` ADD KEY `idx_messages_channel_id_deleted_at_created_at` (`channel_id`, `deleted_at`, `created_at`)").Error
		},
	}
}
