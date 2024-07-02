package migration

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// v16 パーミッション修正
func v36() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "36",
		Migrate: func(db *gorm.DB) error {
			addedRolePermissions := map[string][]string{
				"user": {
					"delete_my_stamp",
				},
				"write": {
					"delete_my_stamp",
				},
			}
			for role, perms := range addedRolePermissions {
				for _, perm := range perms {
					if err := db.Create(&v36RolePermission{Role: role, Permission: perm}).Error; err != nil {
						return err
					}
				}
			}
			return nil
		},
	}
}

type v36RolePermission struct {
	Role       string `gorm:"type:varchar(30);not null;primaryKey"`
	Permission string `gorm:"type:varchar(30);not null;primaryKey"`
}

func (*v36RolePermission) TableName() string {
	return "user_role_permissions"
}
