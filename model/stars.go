package model

import (
	"github.com/satori/go.uuid"
	"time"
)

// Star starの構造体
type Star struct {
	UserID    uuid.UUID `gorm:"type:char(36);not null;primary_key"`
	ChannelID uuid.UUID `gorm:"type:char(36);not null;primary_key"`
	CreatedAt time.Time `gorm:"precision:6;not null"`
}

// TableName dbの名前を指定する
func (star *Star) TableName() string {
	return "stars"
}
