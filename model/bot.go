package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"github.com/gofrs/uuid"
	"strings"
	"time"
)

// BotEvent ボットイベント
type BotEvent string

// String stringにキャスト
func (be BotEvent) String() string {
	return string(be)
}

// BotEvents ボットイベントのセット
type BotEvents map[BotEvent]bool

// Value database/sql/driver.Valuer 実装
func (set BotEvents) Value() (driver.Value, error) {
	return set.String(), nil
}

// Scan database/sql.Scanner 実装
func (set *BotEvents) Scan(src interface{}) error {
	switch s := src.(type) {
	case nil:
		*set = BotEvents{}
		return nil
	case string:
		as := BotEvents{}
		for _, v := range strings.Split(s, " ") {
			if len(v) > 0 {
				as[BotEvent(v)] = true
			}
		}
		*set = as
		return nil
	case []byte:
		as := BotEvents{}
		for _, v := range strings.Split(string(s), " ") {
			if len(v) > 0 {
				as[BotEvent(v)] = true
			}
		}
		*set = as
		return nil
	default:
		return errors.New("failed to scan BotEvents")
	}
}

// String BotEventsをスペース区切りで文字列に出力します
func (set BotEvents) String() string {
	sa := make([]string, 0, len(set))
	for k := range set {
		sa = append(sa, string(k))
	}
	return strings.Join(sa, " ")
}

// Contains 指定したBotEventが含まれているかどうか
func (set BotEvents) Contains(ev BotEvent) bool {
	return set[ev]
}

// MarshalJSON encoding/json.Marshaler 実装
func (set BotEvents) MarshalJSON() ([]byte, error) {
	arr := make([]string, 0, len(set))
	for e := range set {
		arr = append(arr, string(e))
	}
	return json.Marshal(arr)
}

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
	ID                uuid.UUID  `gorm:"type:char(36);not null;primary_key"`
	BotUserID         uuid.UUID  `gorm:"type:char(36);not null;unique"`
	Description       string     `gorm:"type:text;not null"`
	VerificationToken string     `gorm:"type:varchar(30);not null"`
	AccessTokenID     uuid.UUID  `gorm:"type:char(36);not null"`
	PostURL           string     `gorm:"type:text;not null"`
	SubscribeEvents   BotEvents  `gorm:"type:text;not null"`
	Privileged        bool       `gorm:"type:boolean;not null;default:false"`
	State             BotState   `gorm:"type:tinyint;not null;default:0"`
	BotCode           string     `gorm:"type:varchar(30);not null;unique"`
	CreatorID         uuid.UUID  `gorm:"type:char(36);not null"`
	CreatedAt         time.Time  `gorm:"precision:6"`
	UpdatedAt         time.Time  `gorm:"precision:6"`
	DeletedAt         *time.Time `gorm:"precision:6"`
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
	RequestID uuid.UUID `gorm:"type:char(36);not null;primary_key"                json:"requestId"`
	BotID     uuid.UUID `gorm:"type:char(36);not null;index:bot_id_date_time_idx" json:"botId"`
	Event     BotEvent  `gorm:"type:varchar(30);not null"                         json:"event"`
	Body      string    `gorm:"type:text"                                         json:"-"`
	Error     string    `gorm:"type:text"                                         json:"-"`
	Code      int       `gorm:"not null;default:0"                                json:"code"`
	Latency   int64     `gorm:"not null;default:0"                                json:"-"`
	DateTime  time.Time `gorm:"precision:6;index:bot_id_date_time_idx"            json:"dateTime"`
}

// TableName BotEventLogのテーブル名
func (*BotEventLog) TableName() string {
	return "bot_event_logs"
}
