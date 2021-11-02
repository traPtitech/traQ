package migration

import (
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

// v30 bot_event_logsにresultを追加
func v30() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "30",
		Migrate: func(db *gorm.DB) error {
			if err := db.AutoMigrate(&v30BotEventLog{}); err != nil {
				return err
			}
			return db.Exec("UPDATE bot_event_logs SET result = CASE WHEN code = 204 THEN 'ok' WHEN code = -1 THEN 'ne' ELSE 'ng' END").Error
		},
	}
}

type v30BotEventLog struct {
	RequestID uuid.UUID       `gorm:"type:char(36);not null;primaryKey"`
	BotID     uuid.UUID       `gorm:"type:char(36);not null;index:bot_id_date_time_idx"`
	Event     v30BotEventType `gorm:"type:varchar(30);not null"`
	Body      string          `gorm:"type:text"`
	Result    string          `gorm:"type:char(2);not null"`
	Error     string          `gorm:"type:text"`
	Code      int             `gorm:"not null;default:0"`
	Latency   int64           `gorm:"not null;default:0"`
	DateTime  time.Time       `gorm:"precision:6;index:bot_id_date_time_idx"`
}

func (*v30BotEventLog) TableName() string {
	return "bot_event_logs"
}

type v30BotEventType string
