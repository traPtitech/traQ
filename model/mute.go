package model

import (
	"github.com/gofrs/uuid"
)

// Mute ミュートチャンネルのレコード
type Mute struct {
	UserID    uuid.UUID `gorm:"type:char(36);not null;primary_key"`
	ChannelID uuid.UUID `gorm:"type:char(36);not null;primary_key"`
}

// TableName Mute構造体のテーブル名
func (m *Mute) TableName() string {
	return "mutes"
}
