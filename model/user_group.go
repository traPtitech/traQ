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
	CreatedAt   time.Time `gorm:"precision:6"`
	UpdatedAt   time.Time `gorm:"precision:6"`

	Admins  []*UserGroupAdmin  `gorm:"association_autoupdate:false;association_autocreate:false;preload:false;foreignkey:GroupID"`
	Members []*UserGroupMember `gorm:"association_autoupdate:false;association_autocreate:false;preload:false;foreignkey:GroupID"`
}

// TableName UserGroup構造体のテーブル名
func (*UserGroup) TableName() string {
	return "user_groups"
}

func (ug *UserGroup) IsAdmin(uid uuid.UUID) bool {
	for _, admin := range ug.Admins {
		if admin.UserID == uid {
			return true
		}
	}
	return false
}

func (ug *UserGroup) IsMember(uid uuid.UUID) bool {
	for _, admin := range ug.Members {
		if admin.UserID == uid {
			return true
		}
	}
	return false
}

func (ug *UserGroup) AdminIDArray() []uuid.UUID {
	result := make([]uuid.UUID, len(ug.Admins))
	for i, admin := range ug.Admins {
		result[i] = admin.UserID
	}
	return result
}

// UserGroupMember ユーザーグループメンバー構造体
type UserGroupMember struct {
	GroupID uuid.UUID `gorm:"type:char(36);not null;primary_key"`
	UserID  uuid.UUID `gorm:"type:char(36);not null;primary_key"`
	Role    string    `gorm:"type:varchar(100);not null;default:''"`
}

// TableName UserGroupMember構造体のテーブル名
func (*UserGroupMember) TableName() string {
	return "user_group_members"
}

// UserGroupAdmin ユーザーグループ管理者構造体
type UserGroupAdmin struct {
	GroupID uuid.UUID `gorm:"type:char(36);not null;primary_key"`
	UserID  uuid.UUID `gorm:"type:char(36);not null;primary_key"`
}

// TableName UserGroupAdmin構造体のテーブル名
func (*UserGroupAdmin) TableName() string {
	return "user_group_admins"
}
