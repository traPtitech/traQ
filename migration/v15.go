package migration

import (
	"fmt"
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

// v15 外部ログイン機能追加
func v15() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "15",
		Migrate: func(db *gorm.DB) error {
			if err := db.AutoMigrate(&v15ExternalProviderUser{}); err != nil {
				return err
			}

			foreignKeys := [][6]string{
				// table name, constraint name, field name, references, on delete, on update
				{"external_provider_users", "external_provider_users_user_id_users_id_foreign", "user_id", "users(id)", "CASCADE", "CASCADE"},
			}
			for _, c := range foreignKeys {
				if err := db.Exec(fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s ON DELETE %s ON UPDATE %s", c[0], c[1], c[2], c[3], c[4], c[5])).Error; err != nil {
					return err
				}
			}

			uniqueIndexes := [][3]string{
				// table name, index name, field names
				{"external_provider_users", "idx_external_provider_users_provider_name_external_id", "(provider_name, external_id)"},
			}
			for _, c := range uniqueIndexes {
				if err := db.Exec(fmt.Sprintf("ALTER TABLE %s ADD UNIQUE KEY %s %s", c[0], c[1], c[2])).Error; err != nil {
					return err
				}
			}

			return nil
		},
		Rollback: nil,
	}
}

type v15ExternalProviderUser struct {
	UserID       uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
	ProviderName string    `gorm:"type:varchar(30);not null;primaryKey"`
	ExternalID   string    `gorm:"type:varchar(100);not null"`
	Extra        string    `gorm:"type:text;not null"`
	CreatedAt    time.Time `gorm:"precision:6"`
	UpdatedAt    time.Time `gorm:"precision:6"`
}

func (v15ExternalProviderUser) TableName() string {
	return "external_provider_users"
}
