package model

import (
	"time"

	"github.com/gofrs/uuid"
)

// MessageStamp メッセージスタンプ構造体
type MessageStamp struct {
	MessageID uuid.UUID `gorm:"type:char(36);not null;primaryKey;index" json:"-"`
	StampID   uuid.UUID `gorm:"type:char(36);not null;primaryKey;index:idx_messages_stamps_user_id_stamp_id_updated_at,priority:2" json:"stampId"`
	UserID    uuid.UUID `gorm:"type:char(36);not null;primaryKey;index:idx_messages_stamps_user_id_stamp_id_updated_at,priority:1;index:idx_messages_stamps_user_id_updated_at,priority:1" json:"userId"`
	Count     int       `gorm:"type:int;not null" json:"count"`
	CreatedAt time.Time `gorm:"precision:6" json:"createdAt"`
	UpdatedAt time.Time `gorm:"precision:6;index;index:idx_messages_stamps_user_id_stamp_id_updated_at,priority:3;index:idx_messages_stamps_user_id_updated_at,priority:2" json:"updatedAt"`

	Stamp *Stamp `gorm:"constraint:messages_stamps_stamp_id_stamps_id_foreign,OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
	User  *User  `gorm:"constraint:messages_stamps_user_id_users_id_foreign,OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
}

// TableName メッセージスタンプのテーブル
func (*MessageStamp) TableName() string {
	return "messages_stamps"
}
