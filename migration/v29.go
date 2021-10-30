package migration

import (
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

// v29 BotにModeを追加、WebSocket Modeを追加
func v29() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "29",
		Migrate: func(db *gorm.DB) error {
			if err := db.AutoMigrate(&v29Bot{}); err != nil {
				return err
			}
			return db.Exec("ALTER TABLE bots ALTER COLUMN mode DROP DEFAULT").Error
		},
	}
}

type v29Bot struct {
	ID                uuid.UUID        `gorm:"type:char(36);not null;primaryKey"`
	BotUserID         uuid.UUID        `gorm:"type:char(36);not null;unique"`
	Description       string           `gorm:"type:text;not null"`
	VerificationToken string           `gorm:"type:varchar(30);not null"`
	AccessTokenID     uuid.UUID        `gorm:"type:char(36);not null"`
	PostURL           string           `gorm:"type:text;not null"`
	SubscribeEvents   v29BotEventTypes `gorm:"type:text;not null"`
	Privileged        bool             `gorm:"type:boolean;not null;default:false"`
	Mode              v29BotMode       `gorm:"type:varchar(30);not null;default:'HTTP'"` // added, has default value for migration
	State             v29BotState      `gorm:"type:tinyint;not null;default:0"`
	BotCode           string           `gorm:"type:varchar(30);not null;unique"`
	CreatorID         uuid.UUID        `gorm:"type:char(36);not null"`
	CreatedAt         time.Time        `gorm:"precision:6"`
	UpdatedAt         time.Time        `gorm:"precision:6"`
	DeletedAt         gorm.DeletedAt   `gorm:"precision:6"`
}

func (*v29Bot) TableName() string {
	return "bots"
}

type (
	v29BotMode       string
	v29BotState      int
	v29BotEventType  string
	v29BotEventTypes map[v29BotEventType]struct{}
)
