package migration

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// v42 messages_stampsテーブルへの (user_id, updated_at) の複合インデックスの追加
func v42() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "42",
		Migrate: func(db *gorm.DB) error {
			return db.Exec("CREATE INDEX idx_messages_stamps_user_id_updated_at ON messages_stamps (user_id, updated_at)").Error
		},
		Rollback: func(db *gorm.DB) error {
			return db.Exec("DROP INDEX idx_messages_stamps_user_id_updated_at ON messages_stamps").Error
		},
	}
}
