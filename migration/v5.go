package migration

import (
	"fmt"
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

// v5 Mute, 旧Clip削除, stampsにカラム追加
func v5() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "5",
		Migrate: func(db *gorm.DB) error {
			deleteForeignKeys := [][2]string{
				// table name, constraint name
				{"mutes", "mutes_user_id_users_id_foreign"},
				{"mutes", "mutes_channel_id_channels_id_foreign"},
				{"clips", "clips_folder_id_clip_folders_id_foreign"},
				{"clips", "clips_message_id_messages_id_foreign"},
				{"clips", "clips_user_id_users_id_foreign"},
				{"clip_folders", "clip_folders_user_id_users_id_foreign"},
			}
			for _, c := range deleteForeignKeys {
				if err := db.Exec(fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT %s", c[0], c[1])).Error; err != nil {
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

			if err := db.AutoMigrate(&v5Stamp{}); err != nil {
				return err
			}

			return db.Migrator().DropTable("mutes", "clips", "clip_folders")
		},
	}
}

type v5Stamp struct {
	ID        uuid.UUID      `gorm:"type:char(36);not null;primaryKey"`
	Name      string         `gorm:"type:varchar(32);not null;unique"`
	CreatorID uuid.UUID      `gorm:"type:char(36);not null"`
	FileID    uuid.UUID      `gorm:"type:char(36);not null"`
	IsUnicode bool           `gorm:"type:boolean;not null;default:false;index"`
	CreatedAt time.Time      `gorm:"precision:6"`
	UpdatedAt time.Time      `gorm:"precision:6"`
	DeletedAt gorm.DeletedAt `gorm:"precision:6"`
}

func (*v5Stamp) TableName() string {
	return "stamps"
}
