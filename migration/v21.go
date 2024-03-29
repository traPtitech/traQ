package migration

import (
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"

	"github.com/traPtitech/traQ/model"
)

// v21 OGPキャッシュ追加
func v21() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "21",
		Migrate: func(db *gorm.DB) error {
			return db.AutoMigrate(&v21OgpCache{})
		},
	}
}

type v21OgpCache struct {
	ID        int       `gorm:"auto_increment;not null;primaryKey"`
	URL       string    `gorm:"type:text;not null"`
	URLHash   string    `gorm:"type:char(40);not null;index"`
	Valid     bool      `gorm:"type:boolean"`
	Content   model.Ogp `gorm:"type:text"`
	ExpiresAt time.Time `gorm:"precision:6"`
}

func (ogp v21OgpCache) TableName() string {
	return "ogp_cache"
}
