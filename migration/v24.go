package migration

import (
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"gopkg.in/gormigrate.v1"
)

// v24 ユーザー設定追加
func v24() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "24",
		Migrate: func(db *gorm.DB) error {
			// ユーザー設定追加
			if err := db.AutoMigrate(&v24UserSetting{}).Error; err != nil {
				return err
			}
			foreignKeys := [][5]string{
				{"user_settings", "user_id", "users(id)", "CASCADE", "CASCADE"},
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

type v24UserSetting struct {
	UserID         uuid.UUID `gorm:"type:char(36);not null;primary_key"`
	NotifyCitation bool      `gorm:"type:boolean"`
}

func (v24UserSetting) TableName() string {
	return "user_settings"
}
