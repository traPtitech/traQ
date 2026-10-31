package migration

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"

	"github.com/traPtitech/traQ/model"
)

// v44 画像埋め込みジョブテーブルを追加
func v44() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "44",
		Migrate: func(db *gorm.DB) error {
			return db.AutoMigrate(&model.ImageEmbeddingJob{})
		},
		Rollback: func(db *gorm.DB) error {
			return db.Migrator().DropTable(&model.ImageEmbeddingJob{})
		},
	}
}
