package model

import (
	"time"

	"github.com/satori/go.uuid"
)

// Device 通知デバイスの構造体
type Device struct {
	Token     string    `gorm:"type:varchar(190);not null;primary_key"`
	UserID    uuid.UUID `gorm:"type:char(36);not null;index"`
	CreatedAt time.Time `gorm:"precision:6"`
}

// TableName Device構造体のテーブル名
func (*Device) TableName() string {
	return "devices"
}
