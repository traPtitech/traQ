package model

import (
	"time"

	"github.com/gofrs/uuid"
)

// ClipFolders クリップフォルダーの構造体
type ClipFolder struct {
	ID          uuid.UUID `gorm:"type:char(36);not null;primary_key"`
	Name        string    `gorm:"type:varchar(30);not null"`
	Description string    `gorm:"type:text;not null"`
	OwnerID     uuid.UUID `gorm:"type:char(36);not null;index"`
	CreatedAt   time.Time `gorm:"precision:6"`
}

// TableName ClipFolders構造体のテーブル名
func (*ClipFolder) TableName() string {
	return "clip_folders"
}

// ClipFolderMessage クリップフォルダーのメッセージの構造体
type ClipFolderMessage struct {
	FolderID  uuid.UUID `gorm:"type:char(36);not null;primary_key"`
	MessageID uuid.UUID `gorm:"type:char(36);not null;primary_key"`
	ClippedAt time.Time `gorm:"precision:6"`
}

// TableName ClipFolderMessage構造体のテーブル名
func (*ClipFolderMessage) TableName() string {
	return "clip_folder_messages"
}
