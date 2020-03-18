package model

import (
	"time"

	"github.com/gofrs/uuid"
)

// ClipFolders クリップフォルダーの構造体
type ClipFolders struct {
	ID          uuid.UUID `gorm:"type:char(36);not null;primary_key"`
	Name        string    `gorm:"type:varchar(30);not null"`
	Description string    `gorm:"type:text"`
	OwnerID     uuid.UUID `gorm:"type:char(36);not null;index"`
	CreatedAt   time.Time `gorm:"precision:6"`
}

// TableName ClipFolders構造体のテーブル名
func (*ClipFolders) TableName() string {
	return "clip_folders"
}

// ClipFolderMessages クリップフォルダーのメッセージの構造体
type ClipFolderMessages struct {
	FolderID  uuid.UUID `gorm:"type:char(36);not null;primary_key"`
	MessageID uuid.UUID `gorm:"type:char(36);not null;primary_key"`
	ClippedAt time.Time `gorm:"precision:6"`
}

// TableName ClipFolderMessages構造体のテーブル名
func (*ClipFolderMessages) TableName() string {
	return "clip_folder_messages"
}
