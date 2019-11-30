package model

import (
	"database/sql/driver"
	"errors"
	vd "github.com/go-ozzo/ozzo-validation"
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

const (
	// BotEventPing Pingイベント
	BotEventPing BotEvent = "PING"
	// BotEventJoined チャンネル参加イベント
	BotEventJoined BotEvent = "JOINED"
	// BotEventLeft チャンネル退出イベント
	BotEventLeft BotEvent = "LEFT"
	// BotEventMessageCreated メッセージ作成イベント
	BotEventMessageCreated BotEvent = "MESSAGE_CREATED"
	// BotEventMentionMessageCreated メンションメッセージ作成イベント
	BotEventMentionMessageCreated BotEvent = "MENTION_MESSAGE_CREATED"
	// BotEventDirectMessageCreated ダイレクトメッセージ作成イベント
	BotEventDirectMessageCreated BotEvent = "DIRECT_MESSAGE_CREATED"
	// BotEventChannelCreated チャンネル作成イベント
	BotEventChannelCreated BotEvent = "CHANNEL_CREATED"
	// BotEventChannelTopicChanged チャンネルトピック変更イベント
	BotEventChannelTopicChanged BotEvent = "CHANNEL_TOPIC_CHANGED"
	// BotEventUserCreated ユーザー作成イベント
	BotEventUserCreated BotEvent = "USER_CREATED"
	// BotEventStampCreated スタンプ作成イベント
	BotEventStampCreated BotEvent = "STAMP_CREATED"
)

// BotEventSet ボットイベント一覧
var BotEventSet = map[BotEvent]bool{
	BotEventPing:                  true,
	BotEventJoined:                true,
	BotEventLeft:                  true,
	BotEventMessageCreated:        true,
	BotEventMentionMessageCreated: true,
	BotEventDirectMessageCreated:  true,
	BotEventChannelCreated:        true,
	BotEventChannelTopicChanged:   true,
	BotEventUserCreated:           true,
	BotEventStampCreated:          true,
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
	return json.Marshal(set.StringArray())
}

// UnmarshalJSON encoding/json.Unmarshaler 実装
func (set *BotEvents) UnmarshalJSON(data []byte) error {
	var str []string
	err := json.Unmarshal(data, &str)
	if err != nil {
		return err
	}

	s := BotEvents{}
	for _, v := range str {
		s[BotEvent(v)] = true
	}
	*set = s
	return nil
}

// Clone BotEventsを複製します
func (set BotEvents) Clone() BotEvents {
	dst := make(BotEvents, len(set))
	for k, v := range set {
		dst[k] = v
	}
	return dst
}

// StringArray BotEventsをstringの配列に変換します
func (set BotEvents) StringArray() (r []string) {
	r = make([]string, 0, len(set))
	for s := range set {
		r = append(r, s.String())
	}
	return r
}

// Validate github.com/go-ozzo/ozzo-validation.Validatable 実装
func (set BotEvents) Validate() error {
	return vd.Validate(set.StringArray(), vd.Each(vd.Required, vd.By(func(value interface{}) error {
		s, _ := value.(string)
		if !BotEventSet[BotEvent(s)] {
			return errors.New("must be bot event")
		}
		return nil
	})))
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
