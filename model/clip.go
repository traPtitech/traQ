package model

import (
	"time"

	"github.com/gofrs/uuid"
)

// ClipFolder クリップフォルダーの構造体
type ClipFolder struct {
	ID          uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
	Name        string    `gorm:"type:varchar(30);not null"`
	Description string    `gorm:"type:text;not null"`
	OwnerID     uuid.UUID `gorm:"type:char(36);not null;index"`
	CreatedAt   time.Time `gorm:"precision:6"`

	Owner *User `gorm:"constraint:clip_folders_owner_id_users_id_foreign,OnUpdate:CASCADE,OnDelete:CASCADE;foreignKey:OwnerID"`
}

// TableName ClipFolder構造体のテーブル名
func (*ClipFolder) TableName() string {
	return "clip_folders"
}

// ClipFolderMessage クリップフォルダーのメッセージの構造体
type ClipFolderMessage struct {
	FolderID  uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
	MessageID uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
	CreatedAt time.Time `gorm:"precision:6"`

	Folder  ClipFolder `gorm:"constraint:clip_folder_messages_folder_id_clip_folders_id_foreign,OnUpdate:CASCADE,OnDelete:CASCADE;foreignKey:FolderID"`
	Message Message    `gorm:"constraint:clip_folder_messages_message_id_messages_id_foreign,OnUpdate:CASCADE,OnDelete:CASCADE"`
}

// TableName ClipFolderMessage構造体のテーブル名
func (*ClipFolderMessage) TableName() string {
	return "clip_folder_messages"
}
