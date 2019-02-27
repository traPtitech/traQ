package model

import (
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/utils/validator"
	"time"
)

// Tag tag_idの管理をする構造体
type Tag struct {
	ID         uuid.UUID `gorm:"type:char(36);not null;primary_key"`
	Name       string    `gorm:"type:varchar(30);not null;unique"   validate:"required,max=30"`
	Restricted bool      `gorm:"type:boolean;not null;default:false"`
	Type       string    `gorm:"type:varchar(30);not null;default:''"`
	CreatedAt  time.Time `gorm:"precision:6"`
	UpdatedAt  time.Time `gorm:"precision:6"`
}

// TableName DBの名前を指定
func (*Tag) TableName() string {
	return "tags"
}

// Validate 構造体を検証します
func (t *Tag) Validate() error {
	return validator.ValidateStruct(t)
}

// UsersTag userTagの構造体
type UsersTag struct {
	UserID    uuid.UUID `gorm:"type:char(36);not null;primary_key"`
	TagID     uuid.UUID `gorm:"type:char(36);not null;primary_key"`
	Tag       Tag       `gorm:"association_autoupdate:false;association_autocreate:false"`
	IsLocked  bool      `gorm:"type:boolean;not null;default:false"`
	CreatedAt time.Time `gorm:"precision:6;index"`
	UpdatedAt time.Time `gorm:"precision:6"`
}

// TableName DBの名前を指定
func (*UsersTag) TableName() string {
	return "users_tags"
}
