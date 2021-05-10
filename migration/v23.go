package migration

import (
	"fmt"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// v23 複合インデックス追加
func v23() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "23",
		Migrate: func(db *gorm.DB) error {
			// 複合インデックス
			indexes := [][3]string{
				// table name, index name, field names
				{"messages", "idx_messages_deleted_at_updated_at", "(deleted_at, updated_at)"},
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
