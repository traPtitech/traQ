package model

import (
	"github.com/satori/go.uuid"
)

// Mute ミュートチャンネルのレコード
type Mute struct {
	UserID    uuid.UUID `gorm:"type:char(36);primary_key"`
	ChannelID uuid.UUID `gorm:"type:char(36);primary_key"`
}

// TableName Mute構造体のテーブル名
func (m *Mute) TableName() string {
	return "mutes"
}
