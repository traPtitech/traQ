package migration

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// v40 delete_my_stampパーミッション削除
func v40() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "40",
		Migrate: func(db *gorm.DB) error {
			deletedRolePermissions := map[string][]string{
				"user": {
					"delete_my_stamp",
				},
				"write": {
					"delete_my_stamp",
				},
			}
			for role, perms := range deletedRolePermissions {
				for _, perm := range perms {
					if err := db.Delete(&v40RolePermission{Role: role, Permission: perm}).Error; err != nil {
						return err
					}
				}
			}
			return nil
		},
	}
}

type v40RolePermission struct {
	Role       string `gorm:"type:varchar(30);not null;primaryKey"`
	Permission string `gorm:"type:varchar(30);not null;primaryKey"`
}

func (*v40RolePermission) TableName() string {
	return "user_role_permissions"
}
