package migration

import (
	"github.com/jinzhu/gorm"
	"gopkg.in/gormigrate.v1"
)

// V1 インデックスidx_messages_channel_id_deleted_at_created_atの追加
var V1 = &gormigrate.Migration{
	ID: "1",
	Migrate: func(db *gorm.DB) error {
		return db.
			Table("messages").
			AddIndex("idx_messages_channel_id_deleted_at_created_at", "channel_id", "deleted_at", "created_at").
			Error
	},
	Rollback: func(db *gorm.DB) error {
		return db.
			Table("messages").
			RemoveIndex("idx_messages_channel_id_deleted_at_created_at").
			Error
	},
}
