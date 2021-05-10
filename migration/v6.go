package migration

import (
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

// v6 ユーザーグループ拡張
func v6() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "6",
		Migrate: func(db *gorm.DB) error {
			if err := db.AutoMigrate(&v6UserGroupMember{}, &v6UserGroupAdmin{}); err != nil {
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

			if err := db.Migrator().DropColumn(&v6OldUserGroup{}, "admin_user_id"); err != nil {
				return err
			}

			return nil
		},
	}
}

type v6OldUserGroup struct {
	ID          uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
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
	GroupID uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
	UserID  uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
	Role    string    `gorm:"type:varchar(100);not null;default:''"`
}

func (v6UserGroupMember) TableName() string {
	return "user_group_members"
}

type v6UserGroupAdmin struct {
	GroupID uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
	UserID  uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
}

func (v6UserGroupAdmin) TableName() string {
	return "user_group_admins"
}
