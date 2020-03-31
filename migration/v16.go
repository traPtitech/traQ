package migration

import (
	"github.com/jinzhu/gorm"
	"gopkg.in/gormigrate.v1"
)

// v16 パーミッション修正
func v16() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "16",
		Migrate: func(db *gorm.DB) error {
			addedRolePermissions := map[string][]string{
				"user": {
					"get_my_external_account",
					"edit_my_external_account",
				},
			}
			for role, perms := range addedRolePermissions {
				for _, perm := range perms {
					if err := db.Create(&v16RolePermission{Role: role, Permission: perm}).Error; err != nil {
						return err
					}
				}
			}
			return nil
		},
	}
}

type v16RolePermission struct {
	Role       string `gorm:"type:varchar(30);not null;primary_key"`
	Permission string `gorm:"type:varchar(30);not null;primary_key"`
}

func (*v16RolePermission) TableName() string {
	return "user_role_permissions"
}
