package migration

import (
	"github.com/jinzhu/gorm"
	"gopkg.in/gormigrate.v1"
)

// v4 Webhook, Bot外部キー
func v4() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "4",
		Migrate: func(db *gorm.DB) error {
			foreignKeys := [][5]string{
				{"webhook_bots", "creator_id", "users(id)", "CASCADE", "CASCADE"},
				{"webhook_bots", "channel_id", "channels(id)", "CASCADE", "CASCADE"},
				{"bots", "creator_id", "users(id)", "CASCADE", "CASCADE"},
				{"bots", "bot_user_id", "users(id)", "CASCADE", "CASCADE"},
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
