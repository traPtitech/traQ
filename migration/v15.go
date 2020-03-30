package migration

import (
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"gopkg.in/gormigrate.v1"
	"time"
)

// v15 外部ログイン機能追加
func v15() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "15",
		Migrate: func(db *gorm.DB) error {
			if err := db.AutoMigrate(&v15ExternalProviderUser{}).Error; err != nil {
				return err
			}

			foreignKeys := [][5]string{
				{"external_provider_users", "user_id", "users(id)", "CASCADE", "CASCADE"},
			}
			for _, c := range foreignKeys {
				if err := db.Table(c[0]).AddForeignKey(c[1], c[2], c[3], c[4]).Error; err != nil {
					return err
				}
			}

			uniqueIndexes := [][]string{
				{"idx_external_provider_users_provider_name_external_id", "external_provider_users", "provider_name", "external_id"},
			}
			for _, v := range uniqueIndexes {
				if err := db.Table(v[1]).AddUniqueIndex(v[0], v[2:]...).Error; err != nil {
					return err
				}
			}

			return nil
		},
	}
}

type v15ExternalProviderUser struct {
	UserID       uuid.UUID `gorm:"type:char(36);not null;primary_key"`
	ProviderName string    `gorm:"type:varchar(30);not null;primary_key"`
	ExternalID   string    `gorm:"type:varchar(100);not null"`
	Extra        string    `gorm:"type:text;not null"`
	CreatedAt    time.Time `gorm:"precision:6"`
	UpdatedAt    time.Time `gorm:"precision:6"`
}

func (v15ExternalProviderUser) TableName() string {
	return "external_provider_users"
}
