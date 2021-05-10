package migration

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// v10 パーミッション周りの調整
func v10() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "10",
		Migrate: func(db *gorm.DB) error {
			deletedPermissions := []string{
				"install_bot",
				"uninstall_bot",
				"edit_stamp_name",
			}
			for _, v := range deletedPermissions {
				if err := db.Delete(v10RolePermission{}, v10RolePermission{Permission: v}).Error; err != nil {
					return err
				}
			}
			addedRolePermissions := map[string][]string{
				"bot": {
					"edit_me",
					"get_my_stamp_history",
					"change_my_icon",
					"create_user_group",
					"edit_user_group",
					"delete_user_group",
					"upload_file",
					"bot_action_join_channel",
					"bot_action_leave_channel",
				},
				"manage_bot": {
					"bot_action_join_channel",
					"bot_action_leave_channel",
				},
				"user": {
					"bot_action_join_channel",
					"bot_action_leave_channel",
					"web_rtc",
				},
				"read": {
					"get_clip_folder",
					"get_stamp_palette",
				},
				"write": {
					"create_clip_folder",
					"edit_clip_folder",
					"delete_clip_folder",
					"create_stamp_palette",
					"edit_stamp_palette",
					"delete_stamp_palette",
				},
			}
			for role, perms := range addedRolePermissions {
				for _, perm := range perms {
					if err := db.Create(&v10RolePermission{Role: role, Permission: perm}).Error; err != nil {
						return err
					}
				}
			}
			return nil
		},
	}
}

type v10RolePermission struct {
	Role       string `gorm:"type:varchar(30);not null;primaryKey"`
	Permission string `gorm:"type:varchar(30);not null;primaryKey"`
}

func (*v10RolePermission) TableName() string {
	return "user_role_permissions"
}
