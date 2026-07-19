package migration

import (
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
	"gorm.io/gorm"
)

// v43-1 Added TokenType column in order to handle the Firebase Installation ID
func v43_1() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "43-1",
		Migrate: func(db *gorm.DB) error {
			if err := db.Exec("ALTER TABLE devices DROP PRIMARY KEY, ADD COLUMN token_type VARCHAR(16) NOT NULL AFTER token, ADD PRIMARY KEY (token, token_type)").Error; err != nil {
				return err
			}
			return db.AutoMigrate(&v43_1Device{})
		},
	}
}

type v43_1Device struct {
	Token     string                `gorm:"type:varchar(190);not null;primaryKey"`
	TokenType model.DeviceTokenType `gorm:"type:varchar(16);not null;primaryKey"`
	UserID    uuid.UUID             `gorm:"type:char(36);not null;index"`
	CreatedAt time.Time             `gorm:"precision:6"`

	User *model.User `gorm:"constraint:devices_user_id_users_id_foreign,OnUpdate:CASCADE,OnDelete:CASCADE"`
}

func (*v43_1Device) TableName() string {
	return "devices"
}
