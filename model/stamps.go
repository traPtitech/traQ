package model

import (
	"time"

	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

// Stamp スタンプ構造体
type Stamp struct {
	ID        uuid.UUID      `gorm:"type:char(36);not null;primaryKey"         json:"id"`
	Name      string         `gorm:"type:varchar(32);not null;unique"          json:"name"`
	CreatorID uuid.UUID      `gorm:"type:char(36);not null"                    json:"creatorId"`
	FileID    uuid.UUID      `gorm:"type:char(36);not null"                    json:"fileId"`
	IsUnicode bool           `gorm:"type:boolean;not null;default:false;index" json:"isUnicode"`
	CreatedAt time.Time      `gorm:"precision:6"                               json:"createdAt"`
	UpdatedAt time.Time      `gorm:"precision:6"                               json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"precision:6"                               json:"-"`

	Aliases []StampAlias `gorm:"constraint:stamps_id_stamp_aliases_stamp_id_foreign,OnUpdate:CASCADE,OnDelete:CASCADE;foreignKey:StampID" json:"aliases"`
	File    *FileMeta    `gorm:"constraint:stamps_file_id_files_id_foreign,OnUpdate:CASCADE,OnDelete:NO ACTION;foreignKey:FileID" json:"-"`
}

// TableName スタンプテーブル名を取得します
func (*Stamp) TableName() string {
	return "stamps"
}

// IsSystemStamp システムが作成したスタンプかどうか
func (s *Stamp) IsSystemStamp() bool {
	return s.CreatorID == uuid.Nil && s.ID != uuid.Nil && len(s.Name) > 0
}

type StampAlias struct {
	ID        uuid.UUID `gorm:"type:char(36);primaryKey"         json:"id"`
	StampID   uuid.UUID `gorm:"type:char(36);not null"           json:"stampId"`
	Name      string    `gorm:"type:varchar(32);not null;unique" json:"name"`
	CreatorID uuid.UUID `gorm:"type:char(36);not null"           json:"creatorId"`
}
