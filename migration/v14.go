package migration

import (
	"github.com/jinzhu/gorm"
	"gopkg.in/gormigrate.v1"
)

// v14 パーミッション不足修正
func v14() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "14",
		Migrate: func(db *gorm.DB) error {
			addedRolePermissions := map[string][]string{
				"user": {
					"get_clip_folder",
					"get_stamp_palette",
					"create_clip_folder",
					"edit_clip_folder",
					"delete_clip_folder",
					"create_stamp_palette",
					"edit_stamp_palette",
					"delete_stamp_palette",
					"delete_file",
				},
			}
			for role, perms := range addedRolePermissions {
				for _, perm := range perms {
					if err := db.Create(&v14RolePermission{Role: role, Permission: perm}).Error; err != nil {
						return err
					}
				}
			}
			return nil
		},
	}
}

type v14RolePermission struct {
	Role       string `gorm:"type:varchar(30);not null;primary_key"`
	Permission string `gorm:"type:varchar(30);not null;primary_key"`
}

func (*v14RolePermission) TableName() string {
	return "user_role_permissions"
}
