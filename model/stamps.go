package model

import (
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/utils/validator"
	"time"
)

// Stamp スタンプ構造体
type Stamp struct {
	ID        uuid.UUID  `gorm:"type:char(36);primary_key" json:"id"`
	Name      string     `gorm:"type:varchar(32);unique"   json:"name"      validate:"name,required"`
	CreatorID uuid.UUID  `gorm:"type:char(36)"             json:"creatorId"`
	FileID    uuid.UUID  `gorm:"type:char(36)"             json:"fileId"`
	CreatedAt time.Time  `gorm:"precision:6"               json:"createdAt"`
	UpdatedAt time.Time  `gorm:"precision:6"               json:"updatedAt"`
	DeletedAt *time.Time `gorm:"precision:6"               json:"-"`
}

// TableName スタンプテーブル名を取得します
func (*Stamp) TableName() string {
	return "stamps"
}

// Validate 構造体を検証します
func (s *Stamp) Validate() error {
	return validator.ValidateStruct(s)
}
