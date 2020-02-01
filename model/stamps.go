package model

import (
	"github.com/gofrs/uuid"
	"time"
)

// Stamp スタンプ構造体
type Stamp struct {
	ID        uuid.UUID  `gorm:"type:char(36);not null;primary_key"        json:"id"`
	Name      string     `gorm:"type:varchar(32);not null;unique"          json:"name"`
	CreatorID uuid.UUID  `gorm:"type:char(36);not null"                    json:"creatorId"`
	FileID    uuid.UUID  `gorm:"type:char(36);not null"                    json:"fileId"`
	IsUnicode bool       `gorm:"type:boolean;not null;default:false;index" json:"-"`
	CreatedAt time.Time  `gorm:"precision:6"                               json:"createdAt"`
	UpdatedAt time.Time  `gorm:"precision:6"                               json:"updatedAt"`
	DeletedAt *time.Time `gorm:"precision:6"                               json:"-"`
}

// TableName スタンプテーブル名を取得します
func (*Stamp) TableName() string {
	return "stamps"
}

// IsSystemStamp システムが作成したスタンプかどうか
func (s *Stamp) IsSystemStamp() bool {
	return s.CreatorID == uuid.Nil && s.ID != uuid.Nil && len(s.Name) > 0
}
