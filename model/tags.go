package model

import (
	"time"

	"github.com/gofrs/uuid"
)

// Tag tag_idの管理をする構造体
type Tag struct {
	ID        uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
	Name      string    `gorm:"type:VARCHAR(30) COLLATE utf8mb4_bin NOT NULL;uniqueIndex:name"`
	CreatedAt time.Time `gorm:"precision:6"`
	UpdatedAt time.Time `gorm:"precision:6"`
}

// TableName DBの名前を指定
func (*Tag) TableName() string {
	return "tags"
}

// UsersTag userTagの構造体
type UsersTag struct {
	UserID    uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
	TagID     uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
	IsLocked  bool      `gorm:"type:boolean;not null;default:false"`
	CreatedAt time.Time `gorm:"precision:6;index"`
	UpdatedAt time.Time `gorm:"precision:6"`

	User *User `gorm:"constraint:users_tags_user_id_users_id_foreign,OnUpdate:CASCADE,OnDelete:CASCADE"`
	Tag  Tag   `gorm:"constraint:users_tags_tag_id_tags_id_foreign,OnUpdate:CASCADE,OnDelete:CASCADE"`
}

// TableName DBの名前を指定
func (*UsersTag) TableName() string {
	return "users_tags"
}

func (t *UsersTag) GetUserID() uuid.UUID {
	return t.UserID
}

func (t *UsersTag) GetTagID() uuid.UUID {
	return t.TagID
}

func (t *UsersTag) GetTag() string {
	return t.Tag.Name
}

func (t *UsersTag) GetIsLocked() bool {
	return t.IsLocked
}

func (t *UsersTag) GetCreatedAt() time.Time {
	return t.CreatedAt
}

func (t *UsersTag) GetUpdatedAt() time.Time {
	return t.UpdatedAt
}

type UserTag interface {
	GetUserID() uuid.UUID
	GetTagID() uuid.UUID
	GetTag() string
	GetIsLocked() bool
	GetCreatedAt() time.Time
	GetUpdatedAt() time.Time
}
