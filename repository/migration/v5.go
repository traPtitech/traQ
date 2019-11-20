package migration

import (
	"github.com/jinzhu/gorm"
	"gopkg.in/gormigrate.v1"
)

// V5 Mute, 旧Clip削除
var V5 = &gormigrate.Migration{
	ID: "5",
	Migrate: func(db *gorm.DB) error {
		deleteForeignKeys := [][5]string{
			{"mutes", "user_id", "users(id)", "CASCADE", "CASCADE"},
			{"mutes", "channel_id", "channels(id)", "CASCADE", "CASCADE"},
			{"clips", "folder_id", "clip_folders(id)", "CASCADE", "CASCADE"},
			{"clips", "message_id", "messages(id)", "CASCADE", "CASCADE"},
			{"clips", "user_id", "users(id)", "CASCADE", "CASCADE"},
			{"clip_folders", "user_id", "users(id)", "CASCADE", "CASCADE"},
		}
		for _, c := range deleteForeignKeys {
			if err := db.Table(c[0]).RemoveForeignKey(c[1], c[2]).Error; err != nil {
				return err
			}
		}

		deletePermissions := []string{
			"get_channel_mute",
			"edit_channel_mute",
			"get_clip",
			"create_clip",
			"delete_clip",
			"get_clip_folder",
			"create_clip_folder",
			"patch_clip_folder",
			"delete_clip_folder",
		}
		for _, v := range deletePermissions {
			if err := db.Delete(v2RolePermission{}, v2RolePermission{Permission: v}).Error; err != nil {
				return err
			}
		}

		return db.DropTableIfExists("mutes", "clips", "clip_folders").Error
	},
}
