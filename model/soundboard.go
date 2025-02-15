package model

import "github.com/gofrs/uuid"

// SoundboardItem サウンドボードアイテム
type SoundboardItem struct {
	ID        uuid.UUID  `gorm:"type:char(36);not null;primary_key" json:"id"`
	Name      string     `gorm:"type:varchar(32);not null" json:"name"`
	StampID   *uuid.UUID `gorm:"type:char(36)" json:"stampId"`
	CreatorID uuid.UUID  `gorm:"type:char(36);not null" json:"creatorId"`
}

// TableName サウンドボードアイテムテーブル名を取得します
func (*SoundboardItem) TableName() string {
	return "soundboard_items"
}
