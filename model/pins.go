package model

import (
	"time"

	"github.com/gofrs/uuid"
)

// Pin ピン留めのレコード
type Pin struct {
	ID        uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
	MessageID uuid.UUID `gorm:"type:char(36);not null;unique"`
	UserID    uuid.UUID `gorm:"type:char(36);not null"`
	CreatedAt time.Time `gorm:"precision:6"`

	Message Message
	User    *User `gorm:"constraint:pins_user_id_users_id_foreign,OnUpdate:CASCADE,OnDelete:CASCADE"`
}

// TableName ピン留めテーブル名
func (pin *Pin) TableName() string {
	return "pins"
}
