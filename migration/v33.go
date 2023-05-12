package migration

import (
	"fmt"
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

// v33 未読テーブルにチャンネルIDカラムを追加 / インデックス類の更新 / 不要なレコードの削除
func v33() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "33",
		Migrate: func(db *gorm.DB) error {
			// 凍結ユーザー / Botユーザーの未読を削除
			if err := db.Delete(&v33Unread{}, "user_id IN (?)",
				db.Table("users").Where("status = ?", v33UserAccountStatusDeactivated).Or("bot = ?", true).Select("id"),
			).Error; err != nil {
				return err
			}

			// 更新のためindexを削除
			if err := db.Exec("ALTER TABLE unreads DROP CONSTRAINT IF EXISTS unreads_user_id_users_id_foreign").Error; err != nil {
				return err
			}
			if err := db.Exec("ALTER TABLE unreads DROP CONSTRAINT IF EXISTS unreads_message_id_messages_id_foreign").Error; err != nil {
				return err
			}
			if err := db.Exec("ALTER TABLE unreads DROP INDEX IF EXISTS unreads_message_id_messages_id_foreign").Error; err != nil {
				return err
			}
			if err := db.Exec("ALTER TABLE unreads DROP INDEX IF EXISTS `PRIMARY`").Error; err != nil {
				return err
			}

			// マイグレート
			if err := db.AutoMigrate(&v33Unread{}); err != nil {
				return err
			}

			// デフォルト値の削除
			if err := db.Exec("ALTER TABLE unreads ALTER COLUMN channel_id DROP DEFAULT").Error; err != nil {
				return err
			}
			// 実際のチャンネルIDに更新
			if err := db.Session(&gorm.Session{AllowGlobalUpdate: true}).Model(&v33Unread{}).Updates(map[string]any{
				"channel_id": db.Table("messages").Where("messages.id = unreads.message_id").Select("channel_id"),
			}).Error; err != nil {
				return err
			}
			// // 削除されたメッセージの未読を削除
			if err := db.Delete(&v33Unread{}, "channel_id IS NULL").Error; err != nil {
				return err
			}
			// NOT NULL制約を追加
			if err := db.Exec("ALTER TABLE unreads MODIFY COLUMN channel_id char(36) NOT NULL").Error; err != nil {
				return err
			}

			// 主キー
			if err := db.Exec("ALTER TABLE unreads ADD PRIMARY KEY (user_id, channel_id, message_id)").Error; err != nil {
				return err
			}

			// 外部キー制約
			foreignKeys := [][6]string{
				// table name, constraint name, field name, references, on delete, on update
				{"unreads", "unreads_user_id_users_id_foreign", "user_id", "users(id)", "CASCADE", "CASCADE"},
				{"unreads", "unreads_channel_id_channels_id_foreign", "channel_id", "channels(id)", "CASCADE", "CASCADE"},
				{"unreads", "unreads_message_id_messages_id_foreign", "message_id", "messages(id)", "CASCADE", "CASCADE"},
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

// v33UserAccountStatusDeactivated 凍結状態
const v33UserAccountStatusDeactivated = 0

// v33Unread 未読レコード構造体
type v33Unread struct {
	UserID     uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
	ChannelID  uuid.UUID `gorm:"type:char(36);primaryKey;"` // setting NULLABLE for migration
	MessageID  uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
	Noticeable bool      `gorm:"type:boolean;not null;default:false"`
	CreatedAt  time.Time `gorm:"precision:6"`
}

// TableName v33Unread構造体のテーブル名
func (*v33Unread) TableName() string {
	return "unreads"
}
