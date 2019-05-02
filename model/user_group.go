package model

import (
	"github.com/gofrs/uuid"
	"time"
)

// UserGroup ユーザーグループ構造体
type UserGroup struct {
	ID          uuid.UUID `gorm:"type:char(36);not null;primary_key"`
	Name        string    `gorm:"type:varchar(30);not null;unique"`
	Description string    `gorm:"type:text;not null"`
	Type        string    `gorm:"type:varchar(30);not null;default:''"`
	AdminUserID uuid.UUID `gorm:"type:char(36);not null"`
	CreatedAt   time.Time `gorm:"precision:6"`
	UpdatedAt   time.Time `gorm:"precision:6"`
}

// TableName UserGroup構造体のテーブル名
func (*UserGroup) TableName() string {
	return "user_groups"
}

// UserGroupMember ユーザーグループメンバー構造体
type UserGroupMember struct {
	GroupID uuid.UUID `gorm:"type:char(36);not null;primary_key"`
	UserID  uuid.UUID `gorm:"type:char(36);not null;primary_key"`
}

// TableName UserGroupMember構造体のテーブル名
func (*UserGroupMember) TableName() string {
	return "user_group_members"
}
