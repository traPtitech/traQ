package model

import (
	"time"

	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

// Message データベースに格納するmessageの構造体
type Message struct {
	ID        uuid.UUID      `gorm:"type:char(36);not null;primaryKey"`
	UserID    uuid.UUID      `gorm:"type:char(36);not null;"`
	ChannelID uuid.UUID      `gorm:"type:char(36);not null;index:idx_messages_channel_id_deleted_at_created_at,priority:1"`
	Text      string         `gorm:"type:TEXT COLLATE utf8mb4_bin NOT NULL"`
	CreatedAt time.Time      `gorm:"precision:6;index;index:idx_messages_channel_id_deleted_at_created_at,priority:3;index:idx_messages_deleted_at_created_at,priority:2"`
	UpdatedAt time.Time      `gorm:"precision:6;index:idx_messages_deleted_at_updated_at,priority:2"`
	DeletedAt gorm.DeletedAt `gorm:"precision:6;index:idx_messages_channel_id_deleted_at_created_at,priority:2;index:idx_messages_deleted_at_created_at,priority:1;index:idx_messages_deleted_at_updated_at,priority:1"`

	User    *User          `gorm:"constraint:messages_user_id_users_id_foreign,OnUpdate:CASCADE,OnDelete:CASCADE"`
	Channel *Channel       `gorm:"constraint:messages_channel_id_channels_id_foreign,OnUpdate:CASCADE,OnDelete:CASCADE"`
	Stamps  []MessageStamp `gorm:"constraint:messages_stamps_message_id_messages_id_foreign,OnUpdate:CASCADE,OnDelete:CASCADE;foreignkey:MessageID"`
	Pin     *Pin           `gorm:"constraint:pins_message_id_messages_id_foreign,OnUpdate:CASCADE,OnDelete:CASCADE"`
}

// TableName DBの名前を指定するメソッド
func (m Message) TableName() string {
	return "messages"
}

// ChannelLatestMessage チャンネル別最新メッセージ
type ChannelLatestMessage struct {
	ChannelID uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
	MessageID uuid.UUID `gorm:"type:char(36);not null;"`
	DateTime  time.Time `gorm:"precision:6;index"`
}

// TableName テーブル名
func (m *ChannelLatestMessage) TableName() string {
	return "channel_latest_messages"
}

// Unread 未読レコード
type Unread struct {
	UserID     uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
	MessageID  uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
	Noticeable bool      `gorm:"type:boolean;not null;default:false"`
	CreatedAt  time.Time `gorm:"precision:6"`

	User    User    `gorm:"constraint:unreads_user_id_users_id_foreign,OnUpdate:CASCADE,OnDelete:CASCADE"`
	Message Message `gorm:"constraint:unreads_message_id_messages_id_foreign,OnUpdate:CASCADE,OnDelete:CASCADE"`
}

// TableName テーブル名
func (unread *Unread) TableName() string {
	return "unreads"
}

// ArchivedMessage 編集前のアーカイブ化されたメッセージの構造体
type ArchivedMessage struct {
	ID        uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
	MessageID uuid.UUID `gorm:"type:char(36);not null;index"`
	UserID    uuid.UUID `gorm:"type:char(36);not null"`
	Text      string    `gorm:"type:TEXT COLLATE utf8mb4_bin NOT NULL"`
	DateTime  time.Time `gorm:"precision:6"`
}

// TableName ArchivedMessage構造体のテーブル名
func (am *ArchivedMessage) TableName() string {
	return "archived_messages"
}
