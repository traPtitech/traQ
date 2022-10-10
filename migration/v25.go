package migration

import (
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils/optional"
)

// v25 FileMetaにIsAnimatedImageを追加
func v25() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "25",
		Migrate: func(db *gorm.DB) error {
			// FileMetaにIsAnimatedImageを追加
			if err := db.AutoMigrate(&v25FileMeta{}); err != nil {
				return err
			}
			return nil
		},
	}
}

type v25FileMeta struct {
	ID              uuid.UUID              `gorm:"type:char(36);not null;primaryKey"`
	Name            string                 `gorm:"type:text;not null"`
	Mime            string                 `gorm:"type:text;not null"`
	Size            int64                  `gorm:"type:bigint;not null"`
	CreatorID       optional.Of[uuid.UUID] `gorm:"type:char(36)"`
	Hash            string                 `gorm:"type:char(32);not null"`
	Type            model.FileType         `gorm:"type:varchar(30);not null"`
	HasThumbnail    bool                   `gorm:"type:boolean;not null;default:false"`
	ThumbnailMime   optional.Of[string]    `gorm:"type:text"`
	ThumbnailWidth  int                    `gorm:"type:int;not null;default:0"`
	ThumbnailHeight int                    `gorm:"type:int;not null;default:0"`
	IsAnimatedImage bool                   `gorm:"type:boolean;not null;default:false"` // 追加
	ChannelID       optional.Of[uuid.UUID] `gorm:"type:char(36)"`
	CreatedAt       time.Time              `gorm:"precision:6"`
	DeletedAt       gorm.DeletedAt         `gorm:"precision:6"`
}

func (v25FileMeta) TableName() string {
	return "files"
}
