package migration

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

// v37 サウンドボードアイテム追加
func v37() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "37",
		Migrate: func(db *gorm.DB) error {
			return db.AutoMigrate(&v37SoundboardItem{})
		},
	}
}

type v37SoundboardItem struct {
	ID        uuid.UUID  `gorm:"type:char(36);not null;primary_key" json:"id"`
	Name      string     `gorm:"type:varchar(32);not null" json:"name"`
	StampID   *uuid.UUID `gorm:"type:char(36)" json:"stampId"`
	CreatorID uuid.UUID  `gorm:"type:char(36);not null" json:"creatorId"`
}
