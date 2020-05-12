package model

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/service/bot/event"
	"time"
)

// BotState Bot状態
type BotState int

const (
	// BotInactive ボットが無効化されている
	BotInactive BotState = 0
	// BotActive ボットが有効である
	BotActive BotState = 1
	// BotPaused ボットが一時停止されている
	BotPaused BotState = 2
)

// Bot Bot構造体
type Bot struct {
	ID                uuid.UUID   `gorm:"type:char(36);not null;primary_key"`
	BotUserID         uuid.UUID   `gorm:"type:char(36);not null;unique"`
	Description       string      `gorm:"type:text;not null"`
	VerificationToken string      `gorm:"type:varchar(30);not null"`
	AccessTokenID     uuid.UUID   `gorm:"type:char(36);not null"`
	PostURL           string      `gorm:"type:text;not null"`
	SubscribeEvents   event.Types `gorm:"type:text;not null"`
	Privileged        bool        `gorm:"type:boolean;not null;default:false"`
	State             BotState    `gorm:"type:tinyint;not null;default:0"`
	BotCode           string      `gorm:"type:varchar(30);not null;unique"`
	CreatorID         uuid.UUID   `gorm:"type:char(36);not null"`
	CreatedAt         time.Time   `gorm:"precision:6"`
	UpdatedAt         time.Time   `gorm:"precision:6"`
	DeletedAt         *time.Time  `gorm:"precision:6"`
}

// TableName Botのテーブル名
func (*Bot) TableName() string {
	return "bots"
}

// BotJoinChannel Bot参加チャンネル構造体
type BotJoinChannel struct {
	ChannelID uuid.UUID `gorm:"type:char(36);not null;primary_key"`
	BotID     uuid.UUID `gorm:"type:char(36);not null;primary_key"`
}

// TableName BotJoinChannelのテーブル名
func (*BotJoinChannel) TableName() string {
	return "bot_join_channels"
}

// BotEventLog Botイベントログ
type BotEventLog struct {
	RequestID uuid.UUID  `gorm:"type:char(36);not null;primary_key"                json:"requestId"`
	BotID     uuid.UUID  `gorm:"type:char(36);not null;index:bot_id_date_time_idx" json:"botId"`
	Event     event.Type `gorm:"type:varchar(30);not null"                         json:"event"`
	Body      string     `gorm:"type:text"                                         json:"-"`
	Error     string     `gorm:"type:text"                                         json:"-"`
	Code      int        `gorm:"not null;default:0"                                json:"code"`
	Latency   int64      `gorm:"not null;default:0"                                json:"-"`
	DateTime  time.Time  `gorm:"precision:6;index:bot_id_date_time_idx"            json:"dateTime"`
}

// TableName BotEventLogのテーブル名
func (*BotEventLog) TableName() string {
	return "bot_event_logs"
}
