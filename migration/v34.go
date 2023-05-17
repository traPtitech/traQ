package migration

import (
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

// v43 未読テーブルのcreated_atカラムをメッセージテーブルを元に更新
func v34() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "34",
		Migrate: func(db *gorm.DB) error {
			// 未読テーブルのcreated_atを該当メッセージのcreated_atに更新
			return db.Session(&gorm.Session{AllowGlobalUpdate: true}).Model(&v34Unread{}).Updates(map[string]any{
				"created_at": db.Table("messages").Where("messages.id = unreads.message_id").Select("created_at"),
			}).Error
		},
	}
}

// v34Unread 未読レコード構造体
type v34Unread struct {
	UserID     uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
	ChannelID  uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
	MessageID  uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
	Noticeable bool      `gorm:"type:boolean;not null;default:false"`
	CreatedAt  time.Time `gorm:"precision:6"`
}

// TableName v34Unread構造体のテーブル名
func (*v34Unread) TableName() string {
	return "unreads"
}
