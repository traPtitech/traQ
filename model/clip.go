package model

import (
	"time"

	"github.com/gofrs/uuid"
)

// ClipFolders クリップフォルダーの構造体
type ClipFolders struct {
	FolderID    uuid.UUID `gorm:"type:char(36);not null;index"`
	Name        string    `gorm:"type:varchar(30);not null"`
	Description string    `gorm:"type:varchar(1000)"`
	OwnerID     uuid.UUID `gorm:"type:char(36);not null;primary_key"`
	CreatedAt   time.Time `gorm:"precision:6"`
}

// TableName ClipFolders構造体のテーブル名
func (*ClipFolders) TableName() string {
	return "clip_folder"
}

// ClipFolderMessages クリップフォルダーのメッセージの構造体
type ClipFolderMessages struct {
	FolderID  uuid.UUID `gorm:"type:char(36);not null;index"`
	MessageID uuid.UUID `gorm:"type:char(36);not null;"`
	ClippedAt time.Time `gorm:"precision:6"`
}

// TableName ClipFolderMessages構造体のテーブル名
func (*ClipFolderMessages) TableName() string {
	return "clip_folder_messages"
}
