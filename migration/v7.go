package migration

import (
	"fmt"
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"

	"github.com/traPtitech/traQ/utils/optional"
)

// v7 ファイルメタ拡張
func v7() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "7",
		Migrate: func(db *gorm.DB) error {
			if err := db.AutoMigrate(&v7File{}); err != nil {
				return err
			}

			// サムネイル画像があるファイルのthumbnail_mimeを全てimage/pngに
			if err := db.Table(v7File{}.TableName()).Where("has_thumbnail = true").Update("thumbnail_mime", "image/png").Error; err != nil {
				return err
			}

			// uuid.Nilのcreator_idを全てnullに
			if err := db.Table(v7File{}.TableName()).Where("creator_id = '00000000-0000-0000-0000-000000000000'").Update("creator_id", nil).Error; err != nil {
				return err
			}

			// 複合インデックス
			indexes := [][3]string{
				// table name, index name, field names
				{"files", "idx_files_channel_id_created_at", "(channel_id, created_at)"},
			}
			for _, c := range indexes {
				if err := db.Exec(fmt.Sprintf("ALTER TABLE %s ADD KEY %s %s", c[0], c[1], c[2])).Error; err != nil {
					return err
				}
			}

			// 外部キー制約
			foreignKeys := [][6]string{
				// table name, constraint name, field name, references, on delete, on update
				{"files", "files_channel_id_channels_id_foreign", "channel_id", "channels(id)", "SET NULL", "CASCADE"},
				{"files", "files_creator_id_users_id_foreign", "creator_id", "users(id)", "RESTRICT", "CASCADE"},
				{"files_acl", "files_acl_file_id_files_id_foreign", "file_id", "files(id)", "CASCADE", "CASCADE"},
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

type v7File struct {
	ID              uuid.UUID       `gorm:"type:char(36);not null;primaryKey"`
	Name            string          `gorm:"type:text;not null"`
	Mime            string          `gorm:"type:text;not null"`
	Size            int64           `gorm:"type:bigint;not null"`
	CreatorID       optional.UUID   `gorm:"type:char(36)"` // nullable化
	Hash            string          `gorm:"type:char(32);not null"`
	Type            string          `gorm:"type:varchar(30);not null;default:''"`
	HasThumbnail    bool            `gorm:"type:boolean;not null;default:false"`
	ThumbnailMime   optional.String `gorm:"type:text"` // 追加
	ThumbnailWidth  int             `gorm:"type:int;not null;default:0"`
	ThumbnailHeight int             `gorm:"type:int;not null;default:0"`
	ChannelID       optional.UUID   `gorm:"type:char(36)"` // 追加
	CreatedAt       time.Time       `gorm:"precision:6"`
	DeletedAt       gorm.DeletedAt  `gorm:"precision:6"`
}

func (v7File) TableName() string {
	return "files"
}
