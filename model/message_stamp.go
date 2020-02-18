package model

import (
	"github.com/gofrs/uuid"
	"time"
)

// MessageStamp メッセージスタンプ構造体
type MessageStamp struct {
	MessageID uuid.UUID `gorm:"type:char(36);not null;primary_key;index" json:"-"`
	StampID   uuid.UUID `gorm:"type:char(36);not null;primary_key"       json:"stampId"`
	UserID    uuid.UUID `gorm:"type:char(36);not null;primary_key"       json:"userId"`
	Count     int       `gorm:"type:int;not null"                        json:"count"`
	CreatedAt time.Time `gorm:"precision:6"                              json:"createdAt"`
	UpdatedAt time.Time `gorm:"precision:6;index"                        json:"updatedAt"`
}

// TableName メッセージスタンプのテーブル
func (*MessageStamp) TableName() string {
	return "messages_stamps"
}
