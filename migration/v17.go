package migration

import (
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/traPtitech/traQ/utils/optional"
	"gopkg.in/gormigrate.v1"
	"time"
)

// v17 ユーザーホームチャンネル
func v17() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "17",
		Migrate: func(db *gorm.DB) error {
			if err := db.AutoMigrate(&v17UserProfile{}).Error; err != nil {
				return err
			}

			foreignKeys := [][5]string{
				{"user_profiles", "home_channel", "channels(id)", "CASCADE", "CASCADE"},
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

type v17UserProfile struct {
	UserID      uuid.UUID     `gorm:"type:char(36);not null;primary_key"`
	Bio         string        `sql:"type:TEXT COLLATE utf8mb4_bin NOT NULL"`
	TwitterID   string        `gorm:"type:varchar(15);not null;default:''"`
	LastOnline  optional.Time `gorm:"precision:6"`
	HomeChannel optional.UUID `gorm:"type:char(36)"` // 追加
	UpdatedAt   time.Time     `gorm:"precision:6"`
}

func (v17UserProfile) TableName() string {
	return "user_profiles"
}
