package model

import (
	"github.com/gofrs/uuid"
	"time"
)

// Stamp スタンプ構造体
type Stamp struct {
	ID        uuid.UUID  `gorm:"type:char(36);not null;primary_key" json:"id"`
	Name      string     `gorm:"type:varchar(32);not null;unique"   json:"name"`
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

// FavoriteStamp お気に入りスタンプ構造体
type FavoriteStamp struct {
	UserID    uuid.UUID `gorm:"type:char(36);not null;primary_key" json:"-"`
	StampID   uuid.UUID `gorm:"type:char(36);not null;primary_key" json:"stampId"`
	CreatedAt time.Time `gorm:"precision:6"                        json:"createdAt"`
}

// TableName テーブル名を取得します
func (*FavoriteStamp) TableName() string {
	return "favorite_stamps"
}
