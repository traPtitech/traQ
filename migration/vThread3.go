package migration

import (
	"fmt"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

type vThread3UserThreadSubscription struct {
	UserID    uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
	ChannelID uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
	Level     int       `gorm:"type:tinyint;not null"`
}

func (*vThread3UserThreadSubscription) TableName() string {
	return "user_thread_subscriptions"
}

// vThread3 スレッド機能: user_thread_subscriptions テーブル追加
func vThread3() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "thread3",
		Migrate: func(db *gorm.DB) error {
			if err := db.AutoMigrate(&vThread3UserThreadSubscription{}); err != nil {
				return err
			}

			foreignKeys := [][6]string{
				{"user_thread_subscriptions", "user_thread_subscriptions_user_id_users_id_foreign", "user_id", "users(id)", "CASCADE", "CASCADE"},
				{"user_thread_subscriptions", "user_thread_subscriptions_channel_id_channels_id_foreign", "channel_id", "channels(id)", "CASCADE", "CASCADE"},
			}
			for _, c := range foreignKeys {
				if err := db.Exec(fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s ON DELETE %s ON UPDATE %s", c[0], c[1], c[2], c[3], c[4], c[5])).Error; err != nil {
					return err
				}
			}

			return nil
		},
		Rollback: func(db *gorm.DB) error {
			if err := db.Migrator().DropTable("user_thread_subscriptions"); err != nil {
				return err
			}

			return nil
		},
	}
}
