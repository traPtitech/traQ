package migration

import (
	"fmt"
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

// v3 チャンネルイベント履歴
func v3() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "3",
		Migrate: func(db *gorm.DB) error {
			if err := db.AutoMigrate(&v3ChannelEvent{}); err != nil {
				return err
			}

			foreignKeys := [][6]string{
				// table name, constraint name, field name, references, on delete, on update
				{"channel_events", "channel_events_channel_id_channels_id_foreign", "channel_id", "channels(id)", "CASCADE", "CASCADE"},
			}
			for _, c := range foreignKeys {
				if err := db.Exec(fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s ON DELETE %s ON UPDATE %s", c[0], c[1], c[2], c[3], c[4], c[5])).Error; err != nil {
					return err
				}
			}

			indexes := [][3]string{
				// table name, index name, field names
				{"channel_events", "idx_channel_events_channel_id_date_time", "(channel_id, date_time)"},
				{"channel_events", "idx_channel_events_channel_id_event_type_date_time", "(channel_id, event_type, date_time)"},
			}
			for _, c := range indexes {
				if err := db.Exec(fmt.Sprintf("ALTER TABLE %s ADD KEY %s %s", c[0], c[1], c[2])).Error; err != nil {
					return err
				}
			}

			return nil
		},
	}
}

type v3ChannelEvent struct {
	EventID   uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
	ChannelID uuid.UUID `gorm:"type:char(36);not null"`
	EventType string    `gorm:"type:varchar(30);not null;"`
	Detail    string    `gorm:"type:TEXT COLLATE utf8mb4_bin NOT NULL"`
	DateTime  time.Time `gorm:"precision:6"`
}

func (*v3ChannelEvent) TableName() string {
	return "channel_events"
}
