package migration

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

// v8 チャンネル購読拡張
func v8() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "8",
		Migrate: func(db *gorm.DB) error {
			if err := db.AutoMigrate(&v8UserSubscribeChannel{}); err != nil {
				return err
			}

			if err := db.Table(v8UserSubscribeChannel{}.TableName()).Updates(map[string]interface{}{"mark": true, "notify": true}).Error; err != nil {
				return err
			}
			return nil
		},
	}
}

type v8UserSubscribeChannel struct {
	UserID    uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
	ChannelID uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
	Mark      bool      `gorm:"type:boolean;not null;default:false"` // 追加
	Notify    bool      `gorm:"type:boolean;not null;default:false"` // 追加
}

func (v8UserSubscribeChannel) TableName() string {
	return "users_subscribe_channels"
}
