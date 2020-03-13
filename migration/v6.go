package migration

import (
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"gopkg.in/gormigrate.v1"
	"time"
)

// v6 ユーザーグループ拡張
func v6() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "6",
		Migrate: func(db *gorm.DB) error {
			if err := db.AutoMigrate(&v6UserGroupMember{}, &v6UserGroupAdmin{}).Error; err != nil {
				return err
			}

			var oldGroups []v6OldUserGroup
			if err := db.Find(&oldGroups).Error; err != nil {
				return err
			}
			for _, g := range oldGroups {
				if err := db.Create(&v6UserGroupAdmin{GroupID: g.ID, UserID: g.AdminUserID}).Error; err != nil {
					return err
				}
			}

			if err := db.Table(v6OldUserGroup{}.TableName()).DropColumn("admin_user_id").Error; err != nil {
				return err
			}

			return nil
		},
	}
}

type v6OldUserGroup struct {
	ID          uuid.UUID `gorm:"type:char(36);not null;primary_key"`
	Name        string    `gorm:"type:varchar(30);not null;unique"`
	Description string    `gorm:"type:text;not null"`
	Type        string    `gorm:"type:varchar(30);not null;default:''"`
	AdminUserID uuid.UUID `gorm:"type:char(36);not null"`
	CreatedAt   time.Time `gorm:"precision:6"`
	UpdatedAt   time.Time `gorm:"precision:6"`
}

func (v6OldUserGroup) TableName() string {
	return "user_groups"
}

type v6UserGroupMember struct {
	GroupID uuid.UUID `gorm:"type:char(36);not null;primary_key"`
	UserID  uuid.UUID `gorm:"type:char(36);not null;primary_key"`
	Role    string    `gorm:"type:varchar(100);not null;default:''"`
}

func (v6UserGroupMember) TableName() string {
	return "user_group_members"
}

type v6UserGroupAdmin struct {
	GroupID uuid.UUID `gorm:"type:char(36);not null;primary_key"`
	UserID  uuid.UUID `gorm:"type:char(36);not null;primary_key"`
}

func (v6UserGroupAdmin) TableName() string {
	return "user_group_admins"
}
