package migration

import (
	"github.com/jinzhu/gorm"
	"github.com/traPtitech/traQ/model"
	"gopkg.in/gormigrate.v1"
	"time"
)

// v21 OGPキャッシュ追加
func v21() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "21",
		Migrate: func(db *gorm.DB) error {
			if err := db.AutoMigrate(&v21OgpCache{}).Error; err != nil {
				return err
			}
			return nil
		},
	}
}

type v21OgpCache struct {
	Id 		  int 		`gorm:"auto_increment;not null;primary_key"`
	URL       string    `gorm:"type:text;not null"`
	URLHash   string    `gorm:"type:char(40);not null;index"`
	Content   model.Ogp	`gorm:"type:text"`
	ExpiresAt time.Time `gorm:"precision:6"`
}

func (ogp v21OgpCache) TableName() string {
	return "ogp_cache"
}
