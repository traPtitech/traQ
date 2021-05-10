package model

import (
	"time"

	"github.com/gofrs/uuid"
)

// Device 通知デバイスの構造体
type Device struct {
	Token     string    `gorm:"type:varchar(190);not null;primaryKey"`
	UserID    uuid.UUID `gorm:"type:char(36);not null;index"`
	CreatedAt time.Time `gorm:"precision:6"`

	User *User `gorm:"constraint:devices_user_id_users_id_foreign,OnUpdate:CASCADE,OnDelete:CASCADE"`
}

// TableName Device構造体のテーブル名
func (*Device) TableName() string {
	return "devices"
}
