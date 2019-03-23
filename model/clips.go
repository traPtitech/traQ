package model

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/utils/validator"
	"time"
)

// ClipFolder クリップフォルダの構造体
type ClipFolder struct {
	ID        uuid.UUID `gorm:"type:char(36);not null;primary_key"                                            json:"id"`
	UserID    uuid.UUID `gorm:"type:char(36);not null;unique_index:user_folder"                               json:"-"`
	Name      string    `gorm:"type:varchar(30);not null;unique_index:user_folder" validate:"max=30,required" json:"name"`
	CreatedAt time.Time `gorm:"precision:6"                                                                   json:"createdAt"`
	UpdatedAt time.Time `gorm:"precision:6"                                                                   json:"-"`
}

// TableName ClipFolderのテーブル名
func (*ClipFolder) TableName() string {
	return "clip_folders"
}

// Validate 構造体を検証します
func (f *ClipFolder) Validate() error {
	return validator.ValidateStruct(f)
}

// Clip clipの構造体
type Clip struct {
	ID        uuid.UUID `gorm:"type:char(36);not null;primary_key"`
	UserID    uuid.UUID `gorm:"type:char(36);not null;unique_index:user_message"`
	MessageID uuid.UUID `gorm:"type:char(36);not null;unique_index:user_message"`
	Message   Message   `gorm:"association_autoupdate:false;association_autocreate:false"`
	FolderID  uuid.UUID `gorm:"type:char(36);not null"`
	CreatedAt time.Time `gorm:"precision:6"`
	UpdatedAt time.Time `gorm:"precision:6"`
}

// TableName Clipのテーブル名
func (clip *Clip) TableName() string {
	return "clips"
}
