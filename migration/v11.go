package migration

import (
	"fmt"
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

// v11 クリップ機能追加
func v11() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "11",
		Migrate: func(db *gorm.DB) error {
			if err := db.AutoMigrate(&v11ClipFolder{}); err != nil {
				return err
			}

			if err := db.AutoMigrate(&v11ClipFolderMessage{}); err != nil {
				return err
			}

			foreignKeys := [][6]string{
				// table name, constraint name, field name, references, on delete, on update
				{"clip_folders", "clip_folders_owner_id_users_id_foreign", "owner_id", "users(id)", "CASCADE", "CASCADE"},
				{"clip_folder_messages", "clip_folder_messages_folder_id_clip_folders_id_foreign", "folder_id", "clip_folders(id)", "CASCADE", "CASCADE"},
				{"clip_folder_messages", "clip_folder_messages_message_id_messages_id_foreign", "message_id", "messages(id)", "CASCADE", "CASCADE"},
			}

			for _, c := range foreignKeys {
				if err := db.Exec(fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s ON DELETE %s ON UPDATE %s", c[0], c[1], c[2], c[3], c[4], c[5])).Error; err != nil {
					return err
				}
			}

			return nil
		},
	}
}

type v11ClipFolder struct {
	ID          uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
	Name        string    `gorm:"type:varchar(30);not null"`
	Description string    `gorm:"type:text;not null"`
	OwnerID     uuid.UUID `gorm:"type:char(36);not null;index"`
	CreatedAt   time.Time `gorm:"precision:6"`
}

func (*v11ClipFolder) TableName() string {
	return "clip_folders"
}

type v11ClipFolderMessage struct {
	FolderID  uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
	MessageID uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
	CreatedAt time.Time `gorm:"precision:6"`
}

func (*v11ClipFolderMessage) TableName() string {
	return "clip_folder_messages"
}
