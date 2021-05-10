package migration

import (
	"fmt"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
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

			indexes := [][3]string{
				// table name, index name, field names
				{"files", "idx_files_creator_id_created_at", "(creator_id, created_at)"},
				{"messages_stamps", "idx_messages_stamps_user_id_stamp_id_updated_at", "(user_id, stamp_id, updated_at)"},
			}
			for _, c := range indexes {
				if err := db.Exec(fmt.Sprintf("ALTER TABLE %s ADD KEY %s %s", c[0], c[1], c[2])).Error; err != nil {
					return err
				}
			}
			return nil
		},
	}
}

type v13RolePermission struct {
	Role       string `gorm:"type:varchar(30);not null;primaryKey"`
	Permission string `gorm:"type:varchar(30);not null;primaryKey"`
}

func (*v13RolePermission) TableName() string {
	return "user_role_permissions"
}
