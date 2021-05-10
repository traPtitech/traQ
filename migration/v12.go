package migration

import (
	"fmt"
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"

	"github.com/traPtitech/traQ/model"
)

// v12 カスタムスタンプパレット機能追加
func v12() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "12",
		Migrate: func(db *gorm.DB) error {
			if err := db.AutoMigrate(&v12StampPalette{}); err != nil {
				return err
			}

			foreignKeys := [][6]string{
				// table name, constraint name, field name, references, on delete, on update
				{"stamp_palettes", "stamp_palettes_creator_id_users_id_foreign", "creator_id", "users(id)", "CASCADE", "CASCADE"},
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

type v12StampPalette struct {
	ID          uuid.UUID   `gorm:"type:char(36);not null;primaryKey"`
	Name        string      `gorm:"type:varchar(30);not null"`
	Description string      `gorm:"type:text;not null"`
	Stamps      model.UUIDs `gorm:"type:text;not null"`
	CreatorID   uuid.UUID   `gorm:"type:char(36);not null;index"`
	CreatedAt   time.Time   `gorm:"precision:6"`
	UpdatedAt   time.Time   `gorm:"precision:6"`
}

func (*v12StampPalette) TableName() string {
	return "stamp_palettes"
}
