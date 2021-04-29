package migration

import (
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils/optional"
	"gopkg.in/gormigrate.v1"
	"time"
)

func v26() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "26",
		Migrate: func(db *gorm.DB) error {
			// `files_thumbnails`テーブル追加
			if err := db.AutoMigrate(&v26FileThumbnail{}).Error; err != nil {
				return err
			}

			// foreign key追加
			foreignKeys := [][5]string{
				{"files_thumbnails", "file_id", "files(id)", "CASCADE", "CASCADE"},
			}
			for _, c := range foreignKeys {
				if err := db.Table(c[0]).AddForeignKey(c[1], c[2], c[3], c[4]).Error; err != nil {
					return err
				}
			}

			// `files_thumbnails` に `type` を全て 'image' として移行
			if err := db.Exec("INSERT INTO `files_thumbnails` " +
				"(`files_thumbnails`.`file_id`, `files_thumbnails`.`type`, `files_thumbnails`.`mime`, `files_thumbnails`.`width`, `files_thumbnails`.`height`) " +
				"SELECT `files`.`id`, 'image', `files`.`thumbnail_mime`, `files`.`thumbnail_width`, `files`.`thumbnail_height` FROM `files` WHERE `files`.`has_thumbnail` = 1").Error; err != nil {
				return err
			}

			// カラム削除
			if err := db.Table(v26OldFileMeta{}.TableName()).DropColumn("has_thumbnail").Error; err != nil {
				return err
			}
			if err := db.Table(v26OldFileMeta{}.TableName()).DropColumn("thumbnail_mime").Error; err != nil {
				return err
			}
			if err := db.Table(v26OldFileMeta{}.TableName()).DropColumn("thumbnail_width").Error; err != nil {
				return err
			}
			if err := db.Table(v26OldFileMeta{}.TableName()).DropColumn("thumbnail_height").Error; err != nil {
				return err
			}
			return nil
		},
	}
}

type v26OldFileMeta struct {
	ID              uuid.UUID       `gorm:"type:char(36);not null;primary_key"`
	Name            string          `gorm:"type:text;not null"`
	Mime            string          `gorm:"type:text;not null"`
	Size            int64           `gorm:"type:bigint;not null"`
	CreatorID       optional.UUID   `gorm:"type:char(36)"`
	Hash            string          `gorm:"type:char(32);not null"`
	Type            model.FileType  `gorm:"type:varchar(30);not null;default:''"`
	HasThumbnail    bool            `gorm:"type:boolean;not null;default:false"`
	ThumbnailMime   optional.String `gorm:"type:text"`
	ThumbnailWidth  int             `gorm:"type:int;not null;default:0"`
	ThumbnailHeight int             `gorm:"type:int;not null;default:0"`
	ChannelID       optional.UUID   `gorm:"type:char(36)"`
	CreatedAt       time.Time       `gorm:"precision:6"`
	DeletedAt       *time.Time      `gorm:"precision:6"`
}

func (f v26OldFileMeta) TableName() string {
	return "files"
}

// FileThumbnail ファイルのサムネイル情報の構造体
type v26FileThumbnail struct {
	FileID uuid.UUID           `gorm:"type:char(36);not null;primary_key"`
	Type   model.ThumbnailType `gorm:"type:varchar(30);not null;primary_key"`
	Mime   string              `gorm:"type:text;not null"`
	Width  int                 `gorm:"type:int;not null;default:0"`
	Height int                 `gorm:"type:int;not null;default:0"`
}

func (f v26FileThumbnail) TableName() string {
	return "files_thumbnails"
}
