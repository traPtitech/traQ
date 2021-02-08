package migration

import (
	"github.com/jinzhu/gorm"
	"gopkg.in/gormigrate.v1"
)

// v23 複合インデックス追加
func v23() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "23",
		Migrate: func(db *gorm.DB) error {
			// 複合インデックス追加
			return db.
				Table("messages").
				AddIndex("idx_messages_deleted_at_updated_at", "deleted_at", "updated_at").
				Error
		},
	}
}
