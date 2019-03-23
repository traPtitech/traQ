package model

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/utils/validator"
	"time"
)

// Stamp スタンプ構造体
type Stamp struct {
	ID        uuid.UUID  `gorm:"type:char(36);not null;primary_key" json:"id"`
	Name      string     `gorm:"type:varchar(32);not null;unique"   json:"name"      validate:"name,required"`
	CreatorID uuid.UUID  `gorm:"type:char(36);not null"             json:"creatorId"`
	FileID    uuid.UUID  `gorm:"type:char(36);not null"             json:"fileId"`
	CreatedAt time.Time  `gorm:"precision:6"                        json:"createdAt"`
	UpdatedAt time.Time  `gorm:"precision:6"                        json:"updatedAt"`
	DeletedAt *time.Time `gorm:"precision:6"                        json:"-"`
}

// TableName スタンプテーブル名を取得します
func (*Stamp) TableName() string {
	return "stamps"
}

// Validate 構造体を検証します
func (s *Stamp) Validate() error {
	return validator.ValidateStruct(s)
}
