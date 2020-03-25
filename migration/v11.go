package migration

import (
	"time"

	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"gopkg.in/gormigrate.v1"
)

// v11 クリップ機能追加
func v11() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "11",
		Migrate: func(db *gorm.DB) error {
			if err := db.AutoMigrate(&v11ClipFolder{}).Error; err != nil {
				return err
			}

			if err := db.AutoMigrate(&v11ClipFolderMessage{}).Error; err != nil {
				return err
			}

			foreignKeys := [][5]string{
				{"clip_folders", "owner_id", "users(id)", "CASCADE", "CASCADE"},
				{"clip_folder_messages", "folder_id", "clip_folders(id)", "CASCADE", "CASCADE"},
				{"clip_folder_messages", "message_id", "messages(id)", "CASCADE", "CASCADE"},
			}

			for _, c := range foreignKeys {
				if err := db.Table(c[0]).AddForeignKey(c[1], c[2], c[3], c[4]).Error; err != nil {
					return err
				}
			}

			return nil
		},
	}
}

type v11ClipFolder struct {
	ID          uuid.UUID `gorm:"type:char(36);not null;primary_key"`
	Name        string    `gorm:"type:varchar(30);not null"`
	Description string    `gorm:"type:text"`
	OwnerID     uuid.UUID `gorm:"type:char(36);not null;index"`
	CreatedAt   time.Time `gorm:"precision:6"`
}

func (*v11ClipFolder) TableName() string {
	return "clip_folders"
}

type v11ClipFolderMessage struct {
	FolderID  uuid.UUID `gorm:"type:char(36);not null;primary_key"`
	MessageID uuid.UUID `gorm:"type:char(36);not null;primary_key"`
	ClippedAt time.Time `gorm:"precision:6"`
}

func (*v11ClipFolderMessage) TableName() string {
	return "clip_folder_messages"
}
