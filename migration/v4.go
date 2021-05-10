package migration

import (
	"fmt"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// v4 Webhook, Bot外部キー
func v4() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "4",
		Migrate: func(db *gorm.DB) error {
			foreignKeys := [][6]string{
				// table name, constraint name, field name, references, on delete, on update
				{"webhook_bots", "webhook_bots_creator_id_users_id_foreign", "creator_id", "users(id)", "CASCADE", "CASCADE"},
				{"webhook_bots", "webhook_bots_channel_id_channels_id_foreign", "channel_id", "channels(id)", "CASCADE", "CASCADE"},
				{"bots", "bots_creator_id_users_id_foreign", "creator_id", "users(id)", "CASCADE", "CASCADE"},
				{"bots", "bots_bot_user_id_users_id_foreign", "bot_user_id", "users(id)", "CASCADE", "CASCADE"},
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
