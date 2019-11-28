package migration

import (
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"gopkg.in/gormigrate.v1"
	"time"
)

// v3 チャンネルイベント履歴
var v3 = &gormigrate.Migration{
	ID: "3",
	Migrate: func(db *gorm.DB) error {
		if err := db.AutoMigrate(&v3ChannelEvent{}).Error; err != nil {
			return err
		}

		foreignKeys := [][5]string{
			{"channel_events", "channel_id", "channels(id)", "CASCADE", "CASCADE"},
		}
		for _, c := range foreignKeys {
			if err := db.Table(c[0]).AddForeignKey(c[1], c[2], c[3], c[4]).Error; err != nil {
				return err
			}
		}

		indexes := [][]string{
			{"idx_channel_events_channel_id_date_time", "channel_events", "channel_id", "date_time"},
			{"idx_channel_events_channel_id_event_type_date_time", "channel_events", "channel_id", "event_type", "date_time"},
		}
		for _, c := range indexes {
			if err := db.Table(c[1]).AddIndex(c[0], c[2:]...).Error; err != nil {
				return err
			}
		}

		return nil
	},
}

type v3ChannelEvent struct {
	EventID   uuid.UUID `gorm:"type:char(36);not null;primary_key"`
	ChannelID uuid.UUID `gorm:"type:char(36);not null"`
	EventType string    `gorm:"type:varchar(30);not null;"`
	Detail    string    `sql:"type:TEXT COLLATE utf8mb4_bin NOT NULL"`
	DateTime  time.Time `gorm:"precision:6"`
}

func (*v3ChannelEvent) TableName() string {
	return "channel_events"
}
