package migration

import (
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

// v19 httpセッション管理テーブル変更
func v19() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "19",
		Migrate: func(db *gorm.DB) error {
			if err := db.Migrator().DropColumn(v19OldSessionRecord{}, "last_access"); err != nil {
				return err
			}
			if err := db.Migrator().DropColumn(v19OldSessionRecord{}, "last_ip"); err != nil {
				return err
			}
			if err := db.Migrator().DropColumn(v19OldSessionRecord{}, "last_user_agent"); err != nil {
				return err
			}
			return nil
		},
	}
}

type v19OldSessionRecord struct {
	Token         string    `gorm:"type:varchar(50);primaryKey"`
	ReferenceID   uuid.UUID `gorm:"type:char(36);unique"`
	UserID        uuid.UUID `gorm:"type:varchar(36);index"`
	LastAccess    time.Time `gorm:"precision:6"`
	LastIP        string    `gorm:"type:text"`
	LastUserAgent string    `gorm:"type:text"`
	Data          []byte    `gorm:"type:longblob"`
	Created       time.Time `gorm:"precision:6"`
}

func (v19OldSessionRecord) TableName() string {
	return "r_sessions"
}
