package migration

import (
	"fmt"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// v18 インデックス追加
func v18() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "18",
		Migrate: func(db *gorm.DB) error {
			// 複合インデックス
			indexes := [][3]string{
				// table name, index name, field names
				{"channels", "idx_channels_channels_id_is_public_is_forced", "(id, is_public, is_forced)"},
				{"messages", "idx_messages_deleted_at_created_at", "(deleted_at, created_at)"},
			}
			for _, c := range indexes {
				if err := db.Exec(fmt.Sprintf("ALTER TABLE %s ADD KEY %s %s", c[0], c[1], c[2])).Error; err != nil {
					return err
				}
			}
			return nil
		},
	}
}
