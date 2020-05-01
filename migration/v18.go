package migration

import (
	"github.com/jinzhu/gorm"
	"gopkg.in/gormigrate.v1"
)

// v18 インデックス追加
func v18() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "18",
		Migrate: func(db *gorm.DB) error {
			// 複合インデックス
			indexes := [][]string{
				{"idx_channels_channels_id_is_public_is_forced", "channels", "id", "is_public", "is_forced"},
				{"idx_messages_deleted_at_created_at", "messages", "deleted_at", "created_at"},
			}
			for _, c := range indexes {
				if err := db.Table(c[1]).AddIndex(c[0], c[2:]...).Error; err != nil {
					return err
				}
			}
			return nil
		},
	}
}
