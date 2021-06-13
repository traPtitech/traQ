package model

import (
	"time"

	"github.com/gofrs/uuid"
)

// UserGroup ユーザーグループ構造体
type UserGroup struct {
	ID          uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
	Name        string    `gorm:"type:varchar(30);not null;unique"`
	Description string    `gorm:"type:text;not null"`
	Type        string    `gorm:"type:varchar(30);not null;default:''"`
	Icon        uuid.UUID `gorm:"type:char(36)"`
	CreatedAt   time.Time `gorm:"precision:6"`
	UpdatedAt   time.Time `gorm:"precision:6"`

	Admins   []*UserGroupAdmin  `gorm:"constraint:user_group_admins_group_id_user_groups_id_foreign,OnUpdate:CASCADE,OnDelete:CASCADE;foreignKey:GroupID"`
	Members  []*UserGroupMember `gorm:"constraint:user_group_members_group_id_user_groups_id_foreign,OnUpdate:CASCADE,OnDelete:CASCADE;foreignKey:GroupID"`
	IconFile *FileMeta          `gorm:"constraint:user_group_icon_files_id_foreign,OnUpdate:CASCADE,OnDelete:SET NULL;foreignKey:Icon"`
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
	for _, member := range ug.Members {
		if member.UserID == uid {
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
	GroupID uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
	UserID  uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
	Role    string    `gorm:"type:varchar(100);not null;default:''"`
}

// TableName UserGroupMember構造体のテーブル名
func (*UserGroupMember) TableName() string {
	return "user_group_members"
}

// UserGroupAdmin ユーザーグループ管理者構造体
type UserGroupAdmin struct {
	GroupID uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
	UserID  uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
}

// TableName UserGroupAdmin構造体のテーブル名
func (*UserGroupAdmin) TableName() string {
	return "user_group_admins"
}
