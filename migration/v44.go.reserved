package migration

import (
	"fmt"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

type v44Thread struct {
	ChannelID uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
	MessageID uuid.UUID `gorm:"type:char(36);not null;uniqueIndex:idx_threads_message_id"`
}

func (*v44Thread) TableName() string {
	return "threads"
}

// v44 スレッド機能: channels.type 追加・is_public 削除・name を TEXT 化、threads テーブル追加
func v44() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "44",
		Migrate: func(db *gorm.DB) error {
			if err := db.Exec("ALTER TABLE channels ADD COLUMN type ENUM('public','dm','thread') NULL AFTER is_forced").Error; err != nil {
				return err
			}

			if err := db.Exec("UPDATE channels SET type = 'public' WHERE is_public = 1").Error; err != nil {
				return err
			}

			if err := db.Exec("UPDATE channels SET type = 'dm' WHERE is_public = 0").Error; err != nil {
				return err
			}

			if err := db.Exec("ALTER TABLE channels MODIFY COLUMN type ENUM('public','dm','thread') NOT NULL DEFAULT 'public'").Error; err != nil {
				return err
			}

			_ = db.Migrator().DropIndex("channels", "idx_channel_channels_id_is_public_is_forced")

			if err := db.Migrator().DropColumn("channels", "is_public"); err != nil {
				return err
			}

			if err := db.Exec("ALTER TABLE channels ADD KEY idx_channel_channels_id_type_is_forced (id, type, is_forced)").Error; err != nil {
				return err
			}

			if err := db.Migrator().DropIndex("channels", "name_parent"); err != nil {
				return err
			}

			if err := db.Exec("ALTER TABLE channels MODIFY COLUMN name TEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL").Error; err != nil {
				return err
			}

			if err := db.Exec("ALTER TABLE channels ADD UNIQUE KEY name_parent (name(191), parent_id)").Error; err != nil {
				return err
			}

			if err := db.AutoMigrate(&v44Thread{}); err != nil {
				return err
			}

			foreignKeys := [][6]string{
				{"threads", "threads_channel_id_channels_id_foreign", "channel_id", "channels(id)", "CASCADE", "CASCADE"},
				{"threads", "threads_message_id_messages_id_foreign", "message_id", "messages(id)", "CASCADE", "CASCADE"},
			}
			for _, c := range foreignKeys {
				if err := db.Exec(fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s ON DELETE %s ON UPDATE %s", c[0], c[1], c[2], c[3], c[4], c[5])).Error; err != nil {
					return err
				}
			}

			return nil
		},
		Rollback: func(db *gorm.DB) error {
			foreignKeys := []string{
				"threads_channel_id_channels_id_foreign",
				"threads_message_id_messages_id_foreign",
			}
			for _, name := range foreignKeys {
				_ = db.Migrator().DropConstraint("threads", name)
			}

			if err := db.Migrator().DropTable("threads"); err != nil {
				return err
			}

			if err := db.Migrator().DropIndex("channels", "name_parent"); err != nil {
				return err
			}

			if err := db.Exec("ALTER TABLE channels MODIFY COLUMN name VARCHAR(20) NOT NULL").Error; err != nil {
				return err
			}

			if err := db.Exec("ALTER TABLE channels ADD UNIQUE KEY name_parent (name, parent_id)").Error; err != nil {
				return err
			}

			if err := db.Exec("ALTER TABLE channels ADD COLUMN is_public TINYINT(1) NOT NULL DEFAULT 0 AFTER is_forced").Error; err != nil {
				return err
			}

			if err := db.Exec("UPDATE channels SET is_public = 1 WHERE type = 'public'").Error; err != nil {
				return err
			}

			_ = db.Migrator().DropIndex("channels", "idx_channel_channels_id_type_is_forced")

			if err := db.Migrator().DropColumn("channels", "type"); err != nil {
				return err
			}

			if err := db.Exec("ALTER TABLE channels ADD KEY idx_channel_channels_id_is_public_is_forced (id, is_public, is_forced)").Error; err != nil {
				return err
			}

			return nil
		},
	}
}
