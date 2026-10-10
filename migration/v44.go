package migration

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"

	"github.com/traPtitech/traQ/model"
)

func v44() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "44",
		Migrate: func(db *gorm.DB) error {
			return db.AutoMigrate(&model.ScheduledMessage{}, &model.ScheduledMessageFile{})
		},
		Rollback: func(db *gorm.DB) error {
			return db.Migrator().DropTable(&model.ScheduledMessageFile{}, &model.ScheduledMessage{})
		},
	}
}
