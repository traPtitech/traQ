package model

import (
	"github.com/gofrs/uuid"
	"time"
)

type StampPalette struct {
	ID          uuid.UUID `gorm:"type:char(36);not null;primary_key"`
	Name        string    `gorm:"type:varchar(30);not null"`
	Description string    `gorm:"type:varchar(1000)"`
	Stamps      []Stamp
	CreatorID   uuid.UUID  `gorm:"type:char(36);not null"`
	CreatedAt   time.Time  `gorm:"precision:6"`
	DeletedAt   *time.Time `gorm:"precision:6"`
}

// TableName StampPalettes構造体のテーブル名
func (*StampPalette) TableName() string {
	return "stamp_palettes"
}
