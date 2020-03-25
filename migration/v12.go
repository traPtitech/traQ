package migration

import (
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/traPtitech/traQ/model"
	"gopkg.in/gormigrate.v1"
	"time"
)

// v12 カスタムスタンプパレット機能追加
func v12() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "12",
		Migrate: func(db *gorm.DB) error {
			if err := db.AutoMigrate(&v12StampPalette{}).Error; err != nil {
				return err
			}
			return nil
		},
	}
}

type v12StampPalette struct {
	ID          uuid.UUID   `gorm:"type:char(36);not null;primary_key"`
	Name        string      `gorm:"type:varchar(30);not null"`
	Description string      `gorm:"type:text;not null"`
	Stamps      model.UUIDs `gorm:"type:text;not null"`
	CreatorID   uuid.UUID   `gorm:"type:char(36);not null"`
	CreatedAt   time.Time   `gorm:"precision:6"`
	UpdatedAt   time.Time   `gorm:"precision:6"`
}

func (*v12StampPalette) TableName() string {
	return "stamp_palettes"
}
