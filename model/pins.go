package model

import (
	"github.com/satori/go.uuid"
	"time"
)

// Pin ピン留めのレコード
type Pin struct {
	ID        uuid.UUID `gorm:"type:char(36);not null;primary_key"`
	MessageID uuid.UUID `gorm:"type:char(36);not null;unique"`
	Message   Message   `gorm:"association_autoupdate:false;association_autocreate:false"`
	UserID    uuid.UUID `gorm:"type:char(36);not null"`
	CreatedAt time.Time `gorm:"precision:6"`
}

// TableName ピン留めテーブル名
func (pin *Pin) TableName() string {
	return "pins"
}
