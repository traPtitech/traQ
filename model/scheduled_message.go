package model

import (
	"time"

	"github.com/gofrs/uuid"
)

const (
	ScheduledMessagePending  = "pending"
	ScheduledMessageFailed   = "failed"
	ScheduledMessageSent     = "sent"
	ScheduledMessageCanceled = "canceled"
)

// ScheduledMessage is private to its author until delivery.
type ScheduledMessage struct {
	ID           uuid.UUID              `gorm:"type:char(36);primaryKey"`
	UserID       uuid.UUID              `gorm:"type:char(36);not null;index"`
	ChannelID    uuid.UUID              `gorm:"type:char(36);not null"`
	Content      string                 `gorm:"type:TEXT COLLATE utf8mb4_bin NOT NULL"`
	DraftContent string                 `gorm:"type:TEXT COLLATE utf8mb4_bin NOT NULL"`
	ScheduledAt  time.Time              `gorm:"precision:6;not null;index:idx_scheduled_messages_due,priority:2"`
	Status       string                 `gorm:"type:varchar(16);not null;index:idx_scheduled_messages_due,priority:1"`
	Failure      string                 `gorm:"type:varchar(64);not null"`
	CreatedAt    time.Time              `gorm:"precision:6"`
	User         *User                  `gorm:"constraint:scheduled_messages_user_id_users_id_foreign,OnUpdate:CASCADE,OnDelete:CASCADE"`
	Channel      *Channel               `gorm:"constraint:scheduled_messages_channel_id_channels_id_foreign,OnUpdate:CASCADE,OnDelete:CASCADE"`
	Attachments  []ScheduledMessageFile `gorm:"foreignKey:ScheduledMessageID;constraint:scheduled_message_files_scheduled_message_id_foreign,OnUpdate:CASCADE,OnDelete:CASCADE"`
}

func (ScheduledMessage) TableName() string { return "scheduled_messages" }

// Each private upload can belong to only one reservation, including after cancellation.
type ScheduledMessageFile struct {
	ScheduledMessageID uuid.UUID `gorm:"type:char(36);primaryKey"`
	// Retain the ID when the file is deleted, so delivery detects a missing upload.
	FileID   uuid.UUID `gorm:"type:char(36);primaryKey;uniqueIndex"`
	Position int       `gorm:"not null"`
}

func (ScheduledMessageFile) TableName() string { return "scheduled_message_files" }

func (m *ScheduledMessage) FileIDs() []uuid.UUID {
	ids := make([]uuid.UUID, len(m.Attachments))
	for i, f := range m.Attachments {
		ids[i] = f.FileID
	}
	return ids
}
