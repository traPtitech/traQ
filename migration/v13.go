package migration

import (
	"github.com/jinzhu/gorm"
	"gopkg.in/gormigrate.v1"
)

// v13 パーミッション調整・インデックス付与
func v13() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "13",
		Migrate: func(db *gorm.DB) error {
			addedRolePermissions := map[string][]string{
				"bot": {
					"delete_file",
				},
				"write": {
					"delete_file",
				},
			}
			for role, perms := range addedRolePermissions {
				for _, perm := range perms {
					if err := db.Create(&v13RolePermission{Role: role, Permission: perm}).Error; err != nil {
						return err
					}
				}
			}

			indexes := [][]string{
				{"idx_files_creator_id_created_at", "files", "creator_id", "created_at"},
				{"idx_messages_stamps_user_id_stamp_id_updated_at", "messages_stamps", "user_id", "stamp_id", "updated_at"},
			}
			for _, v := range indexes {
				if err := db.Table(v[1]).AddIndex(v[0], v[2:]...).Error; err != nil {
					return err
				}
			}
			return nil
		},
	}
}

type v13RolePermission struct {
	Role       string `gorm:"type:varchar(30);not null;primary_key"`
	Permission string `gorm:"type:varchar(30);not null;primary_key"`
}

func (*v13RolePermission) TableName() string {
	return "user_role_permissions"
}
