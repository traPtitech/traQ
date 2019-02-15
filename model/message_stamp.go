package model

import (
	"github.com/satori/go.uuid"
	"time"
)

// MessageStamp メッセージスタンプ構造体
type MessageStamp struct {
	MessageID uuid.UUID `gorm:"type:char(36);primary_key" json:"-"`
	StampID   uuid.UUID `gorm:"type:char(36);primary_key" json:"stampId"`
	UserID    uuid.UUID `gorm:"type:char(36);primary_key" json:"userId"`
	Count     int       `                                 json:"count"`
	CreatedAt time.Time `gorm:"precision:6"               json:"createdAt"`
	UpdatedAt time.Time `gorm:"precision:6;index"         json:"updatedAt"`
}

// TableName メッセージスタンプのテーブル
func (*MessageStamp) TableName() string {
	return "messages_stamps"
}

// UserStampHistory スタンプ履歴構造体
type UserStampHistory struct {
	StampID  uuid.UUID `json:"stampId"`
	Datetime time.Time `json:"datetime"`
}
