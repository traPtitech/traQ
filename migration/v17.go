package migration

import (
	"fmt"
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"

	"github.com/traPtitech/traQ/utils/optional"
)

// v17 ユーザーホームチャンネル
func v17() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "17",
		Migrate: func(db *gorm.DB) error {
			if err := db.AutoMigrate(&v17UserProfile{}); err != nil {
				return err
			}

			foreignKeys := [][6]string{
				// table name, constraint name, field name, references, on delete, on update
				{"user_profiles", "user_profiles_home_channel_channels_id_foreign", "home_channel", "channels(id)", "CASCADE", "CASCADE"},
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

type v17UserProfile struct {
	UserID      uuid.UUID     `gorm:"type:char(36);not null;primaryKey"`
	Bio         string        `gorm:"type:TEXT COLLATE utf8mb4_bin NOT NULL"`
	TwitterID   string        `gorm:"type:varchar(15);not null;default:''"`
	LastOnline  optional.Time `gorm:"precision:6"`
	HomeChannel optional.UUID `gorm:"type:char(36)"` // 追加
	UpdatedAt   time.Time     `gorm:"precision:6"`
}

func (v17UserProfile) TableName() string {
	return "user_profiles"
}
