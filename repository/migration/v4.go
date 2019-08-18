package migration

import (
	"github.com/jinzhu/gorm"
	"gopkg.in/gormigrate.v1"
)

// V4 Webhook, Bot外部キー
var V4 = &gormigrate.Migration{
	ID: "4",
	Migrate: func(db *gorm.DB) error {
		for _, c := range v4ForeignKeys() {
			if err := db.Table(c[0]).AddForeignKey(c[1], c[2], c[3], c[4]).Error; err != nil {
				return err
			}
		}
		return nil
	},
	Rollback: func(db *gorm.DB) error {
		for _, c := range v4ForeignKeys() {
			if err := db.Table(c[0]).RemoveForeignKey(c[1], c[2]).Error; err != nil {
				return err
			}
		}
		return nil
	},
}

func v4ForeignKeys() [][5]string {
	return [][5]string{
		{"webhook_bots", "creator_id", "users(id)", "CASCADE", "CASCADE"},
		{"webhook_bots", "channel_id", "channels(id)", "CASCADE", "CASCADE"},
		{"bots", "creator_id", "users(id)", "CASCADE", "CASCADE"},
		{"bots", "bot_user_id", "users(id)", "CASCADE", "CASCADE"},
	}
}
