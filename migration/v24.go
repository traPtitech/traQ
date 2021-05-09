package migration

import (
	"fmt"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

// v24 ユーザー設定追加
func v24() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "24",
		Migrate: func(db *gorm.DB) error {
			// ユーザー設定追加
			if err := db.AutoMigrate(&v24UserSettings{}); err != nil {
				return err
			}

			foreignKeys := [][6]string{
				// table name, constraint name, field name, references, on delete, on update
				{"user_settings", "user_settings_user_id_users_id_foreign", "user_id", "users(id)", "CASCADE", "CASCADE"},
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

type v24UserSettings struct {
	UserID         uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
	NotifyCitation bool      `gorm:"type:boolean;not null;default:false"`
}

func (*v24UserSettings) TableName() string {
	return "user_settings"
}
