package model

import (
	"time"

	"github.com/gofrs/uuid"
)

type StampPalette struct {
	ID          uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
	Name        string    `gorm:"type:varchar(30);not null"`
	Description string    `gorm:"type:text;not null"`
	Stamps      UUIDs     `gorm:"type:text;not null"`
	CreatorID   uuid.UUID `gorm:"type:char(36);not null;index"`
	CreatedAt   time.Time `gorm:"precision:6"`
	UpdatedAt   time.Time `gorm:"precision:6"`

	Creator User `gorm:"constraint:stamp_palettes_creator_id_users_id_foreign,OnUpdate:CASCADE,OnDelete:CASCADE;foreignKey:CreatorID"`
}

// TableName StampPalettes構造体のテーブル名
func (*StampPalette) TableName() string {
	return "stamp_palettes"
}
