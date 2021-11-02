package model

import (
	"database/sql/driver"
	"errors"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/json-iterator/go"
	"gorm.io/gorm"
)

// BotMode Bot動作モード
type BotMode string

const (
	// BotModeHTTP HTTP Mode
	BotModeHTTP BotMode = "HTTP"
	// BotModeWebSocket WebSocket Mode
	BotModeWebSocket BotMode = "WebSocket"
)

func (m BotMode) String() string {
	return string(m)
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
	ID                uuid.UUID      `gorm:"type:char(36);not null;primaryKey"`
	BotUserID         uuid.UUID      `gorm:"type:char(36);not null;unique"`
	Description       string         `gorm:"type:text;not null"`
	VerificationToken string         `gorm:"type:varchar(30);not null"`
	AccessTokenID     uuid.UUID      `gorm:"type:char(36);not null"`
	PostURL           string         `gorm:"type:text;not null"`
	SubscribeEvents   BotEventTypes  `gorm:"type:text;not null"`
	Privileged        bool           `gorm:"type:boolean;not null;default:false"`
	Mode              BotMode        `gorm:"type:varchar(30);not null"`
	State             BotState       `gorm:"type:tinyint;not null;default:0"`
	BotCode           string         `gorm:"type:varchar(30);not null;unique"`
	CreatorID         uuid.UUID      `gorm:"type:char(36);not null"`
	CreatedAt         time.Time      `gorm:"precision:6"`
	UpdatedAt         time.Time      `gorm:"precision:6"`
	DeletedAt         gorm.DeletedAt `gorm:"precision:6"`

	BotUser *User `gorm:"constraint:bots_bot_user_id_users_id_foreign,OnUpdate:CASCADE,OnDelete:CASCADE;foreignKey:BotUserID"`
	Creator *User `gorm:"constraint:bots_creator_id_users_id_foreign,OnUpdate:CASCADE,OnDelete:CASCADE;foreignKey:CreatorID"`
}

// TableName Botのテーブル名
func (*Bot) TableName() string {
	return "bots"
}

// BotJoinChannel Bot参加チャンネル構造体
type BotJoinChannel struct {
	ChannelID uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
	BotID     uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
}

// TableName BotJoinChannelのテーブル名
func (*BotJoinChannel) TableName() string {
	return "bot_join_channels"
}

// BotEventLog Botイベントログ
type BotEventLog struct {
	RequestID uuid.UUID    `gorm:"type:char(36);not null;primaryKey"`
	BotID     uuid.UUID    `gorm:"type:char(36);not null;index:bot_id_date_time_idx"`
	Event     BotEventType `gorm:"type:varchar(30);not null"`
	Body      string       `gorm:"type:text"`
	Result    string       `gorm:"type:char(2);not null"`
	Error     string       `gorm:"type:text"`
	Code      int          `gorm:"not null;default:0"`
	Latency   int64        `gorm:"not null;default:0"`
	DateTime  time.Time    `gorm:"precision:6;index:bot_id_date_time_idx"`
}

// TableName BotEventLogのテーブル名
func (*BotEventLog) TableName() string {
	return "bot_event_logs"
}

// BotEventType Botイベントタイプ
type BotEventType string

func (t BotEventType) String() string {
	return string(t)
}

// BotEventTypes BotイベントタイプのSet
type BotEventTypes map[BotEventType]struct{}

func BotEventTypesFromArray(arr []string) BotEventTypes {
	res := BotEventTypes{}
	for _, v := range arr {
		if len(v) > 0 {
			res[BotEventType(v)] = struct{}{}
		}
	}
	return res
}

// String event.Typesをスペース区切りで文字列に出力します
func (set BotEventTypes) String() string {
	sa := make([]string, 0, len(set))
	for k := range set {
		sa = append(sa, string(k))
	}
	return strings.Join(sa, " ")
}

// Contains 指定したevent.Typeが含まれているかどうか
func (set BotEventTypes) Contains(ev BotEventType) bool {
	_, ok := set[ev]
	return ok
}

// Array event.Typesをstringの配列に変換します
func (set BotEventTypes) Array() (r []string) {
	r = make([]string, 0, len(set))
	for s := range set {
		r = append(r, s.String())
	}
	return r
}

// Clone event.Typesを複製します
func (set BotEventTypes) Clone() BotEventTypes {
	dst := make(BotEventTypes, len(set))
	for k, v := range set {
		dst[k] = v
	}
	return dst
}

// MarshalJSON encoding/json.Marshaler 実装
func (set BotEventTypes) MarshalJSON() ([]byte, error) {
	return jsoniter.ConfigFastest.Marshal(set.Array())
}

// UnmarshalJSON encoding/json.Unmarshaler 実装
func (set *BotEventTypes) UnmarshalJSON(data []byte) error {
	var arr []string
	err := jsoniter.ConfigFastest.Unmarshal(data, &arr)
	if err != nil {
		return err
	}
	*set = BotEventTypesFromArray(arr)
	return nil
}

// Value database/sql/driver.Valuer 実装
func (set BotEventTypes) Value() (driver.Value, error) {
	return set.String(), nil
}

// Scan database/sql.Scanner 実装
func (set *BotEventTypes) Scan(src interface{}) error {
	switch s := src.(type) {
	case nil:
		*set = BotEventTypes{}
	case string:
		*set = BotEventTypesFromArray(strings.Split(s, " "))
	case []byte:
		*set = BotEventTypesFromArray(strings.Split(string(s), " "))
	default:
		return errors.New("failed to scan BotEvents")
	}
	return nil
}
