package model

import (
	"github.com/gofrs/uuid"
)

// Star starの構造体
type Star struct {
	UserID    uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
	ChannelID uuid.UUID `gorm:"type:char(36);not null;primaryKey"`

	User    *User    `gorm:"constraint:stars_user_id_users_id_foreign,OnUpdate:CASCADE,OnDelete:CASCADE"`
	Channel *Channel `gorm:"constraint:stars_channel_id_channels_id_foreign,OnUpdate:CASCADE,OnDelete:CASCADE"`
}

// TableName dbの名前を指定する
func (star *Star) TableName() string {
	return "stars"
}
