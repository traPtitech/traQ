package migration

import (
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"gopkg.in/gormigrate.v1"
	"time"
)

// v19 httpセッション管理テーブル変更
func v19() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "19",
		Migrate: func(db *gorm.DB) error {
			if err := db.Table(v19OldSessionRecord{}.TableName()).DropColumn("last_access").Error; err != nil {
				return err
			}
			if err := db.Table(v19OldSessionRecord{}.TableName()).DropColumn("last_ip").Error; err != nil {
				return err
			}
			if err := db.Table(v19OldSessionRecord{}.TableName()).DropColumn("last_user_agent").Error; err != nil {
				return err
			}
			return nil
		},
	}
}

type v19OldSessionRecord struct {
	Token         string    `gorm:"type:varchar(50);primary_key"`
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
