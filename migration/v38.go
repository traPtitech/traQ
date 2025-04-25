package migration

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// v38 v37で作ったサウンドボードアイテムのテーブル名変更
func v38() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "38",
		Migrate: func(db *gorm.DB) error {
			return db.Migrator().RenameTable(&v37SoundboardItem{}, "soundboard_items")
		},
	}
}
