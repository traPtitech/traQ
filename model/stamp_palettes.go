package model

import (
	"github.com/gofrs/uuid"
	"time"
)

type StampPalette struct {
	ID          uuid.UUID `gorm:"type:char(36);not null;primary_key" json:"id"`
	Name        string    `gorm:"type:varchar(30);not null" json:"name"`
	Description string    `gorm:"type:text;not null" json:"description"`
	Stamps      UUIDs     `gorm:"type:text;not null" json:"stamps"`
	CreatorID   uuid.UUID `gorm:"type:char(36);not null;index" json:"creatorId"`
	CreatedAt   time.Time `gorm:"precision:6" json:"createdAt"`
	UpdatedAt   time.Time `gorm:"precision:6" json:"updatedAt"`
}

// TableName StampPalettes構造体のテーブル名
func (*StampPalette) TableName() string {
	return "stamp_palettes"
}
