package migration

import (
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"gopkg.in/gormigrate.v1"
	"gopkg.in/guregu/null.v3"
	"time"
)

// v7 ファイルメタ拡張
func v7() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "7",
		Migrate: func(db *gorm.DB) error {
			if err := db.Table(v7File{}.TableName()).ModifyColumn("creator_id", "char(36)").Error; err != nil {
				return err
			}

			if err := db.AutoMigrate(&v7File{}).Error; err != nil {
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
			indexes := [][]string{
				{"idx_files_channel_id_created_at", "files", "channel_id", "created_at"},
			}
			for _, c := range indexes {
				if err := db.Table(c[1]).AddIndex(c[0], c[2:]...).Error; err != nil {
					return err
				}
			}

			// 外部キー制約
			foreignKeys := [][5]string{
				{"files", "channel_id", "channels(id)", "SET NULL", "CASCADE"},
				{"files", "creator_id", "users(id)", "RESTRICT", "CASCADE"},
				{"files_acl", "file_id", "files(id)", "CASCADE", "CASCADE"},
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

type v7File struct {
	ID              uuid.UUID     `gorm:"type:char(36);not null;primary_key"`
	Name            string        `gorm:"type:text;not null"`
	Mime            string        `gorm:"type:text;not null"`
	Size            int64         `gorm:"type:bigint;not null"`
	CreatorID       uuid.NullUUID `gorm:"type:char(36)"` // nullable化
	Hash            string        `gorm:"type:char(32);not null"`
	Type            string        `gorm:"type:varchar(30);not null;default:''"`
	HasThumbnail    bool          `gorm:"type:boolean;not null;default:false"`
	ThumbnailMime   null.String   `gorm:"type:text"` // 追加
	ThumbnailWidth  int           `gorm:"type:int;not null;default:0"`
	ThumbnailHeight int           `gorm:"type:int;not null;default:0"`
	ChannelID       uuid.NullUUID `gorm:"type:char(36)"` // 追加
	CreatedAt       time.Time     `gorm:"precision:6"`
	DeletedAt       *time.Time    `gorm:"precision:6"`
}

func (v7File) TableName() string {
	return "files"
}
