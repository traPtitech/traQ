package model

import (
	"database/sql/driver"
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
func (arr BotEvents) Value() (driver.Value, error) {
	return arr.String(), nil
}

// Scan database/sql.Scanner 実装
func (arr *BotEvents) Scan(src interface{}) error {
	if src == nil {
		*arr = BotEvents{}
		return nil
	}
	if sv, err := driver.String.ConvertValue(src); err == nil {
		if v, ok := sv.(string); ok {
			as := BotEvents{}
			for _, v := range strings.Split(v, " ") {
				as[BotEvent(v)] = true
			}
			*arr = as
			return nil
		} else if v, ok := sv.([]byte); ok {
			as := BotEvents{}
			for _, v := range strings.Split(string(v), " ") {
				as[BotEvent(v)] = true
			}
			*arr = as
			return nil
		}
	}
	return errors.New("failed to scan BotEvents")
}

// String BotEventsをスペース区切りで文字列に出力します
func (arr BotEvents) String() string {
	sa := make([]string, 0, len(arr))
	for k := range arr {
		sa = append(sa, string(k))
	}
	return strings.Join(sa, " ")
}

// Contains 指定したBotEventが含まれているかどうか
func (arr BotEvents) Contains(ev BotEvent) bool {
	return arr[ev]
}

// BotStatus Bot状態
type BotStatus int

const (
	// BotInactive ボットが無効化されている
	BotInactive BotStatus = 0
	// BotActive ボットが有効である
	BotActive BotStatus = 1
	// BotPaused ボットが一時停止されている
	BotPaused BotStatus = 2
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
	Status            BotStatus  `gorm:"type:tinyint;not null;default:0"`
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
