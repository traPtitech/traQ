package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"github.com/gofrs/uuid"
	"time"
)

const (
	// DirectMessageChannelRootID ダイレクトメッセージチャンネルの親チャンネルID
	DirectMessageChannelRootID = "aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa"
	// MaxChannelDepth チャンネルの深さの最大
	MaxChannelDepth = 5
)

var dmChannelRootUUID = uuid.Must(uuid.FromString(DirectMessageChannelRootID))

// Channel チャンネルの構造体
type Channel struct {
	ID        uuid.UUID  `gorm:"type:char(36);not null;primary_key"`
	Name      string     `gorm:"type:varchar(20);not null;unique_index:name_parent" validate:"channel,required"`
	ParentID  uuid.UUID  `gorm:"type:char(36);not null;unique_index:name_parent"`
	Topic     string     `sql:"type:TEXT COLLATE utf8mb4_bin NOT NULL"`
	IsForced  bool       `gorm:"type:boolean;not null;default:false"`
	IsPublic  bool       `gorm:"type:boolean;not null;default:false"`
	IsVisible bool       `gorm:"type:boolean;not null;default:false"`
	CreatorID uuid.UUID  `gorm:"type:char(36);not null"`
	UpdaterID uuid.UUID  `gorm:"type:char(36);not null"`
	CreatedAt time.Time  `gorm:"precision:6"`
	UpdatedAt time.Time  `gorm:"precision:6"`
	DeletedAt *time.Time `gorm:"precision:6"`
}

// TableName テーブル名を指定するメソッド
func (ch *Channel) TableName() string {
	return "channels"
}

// IsDMChannel ダイレクトメッセージ用チャンネルかどうかを返します
func (ch *Channel) IsDMChannel() bool {
	return ch.ParentID == dmChannelRootUUID
}

// UsersPrivateChannel UsersPrivateChannelsの構造体
type UsersPrivateChannel struct {
	UserID    uuid.UUID `gorm:"type:char(36);not null;primary_key"`
	ChannelID uuid.UUID `gorm:"type:char(36);not null;primary_key"`
}

// TableName テーブル名を指定するメソッド
func (upc *UsersPrivateChannel) TableName() string {
	return "users_private_channels"
}

// UserSubscribeChannel ユーザー・通知チャンネル対構造体
type UserSubscribeChannel struct {
	UserID    uuid.UUID `gorm:"type:char(36);not null;primary_key"`
	ChannelID uuid.UUID `gorm:"type:char(36);not null;primary_key"`
}

// TableName UserNotifiedChannel構造体のテーブル名
func (*UserSubscribeChannel) TableName() string {
	return "users_subscribe_channels"
}

// DMChannelMapping ダイレクトメッセージチャンネルとユーザーのマッピング
type DMChannelMapping struct {
	ChannelID uuid.UUID `gorm:"type:char(36);not null;primary_key"`
	User1     uuid.UUID `gorm:"type:char(36);not null;unique_index:user1_user2"`
	User2     uuid.UUID `gorm:"type:char(36);not null;unique_index:user1_user2"`
}

// TableName DMChannelMapping構造体のテーブル名
func (*DMChannelMapping) TableName() string {
	return "dm_channel_mappings"
}

// ChannelEventType チャンネルイベントタイプ
type ChannelEventType string

// String stringに変換します
func (t ChannelEventType) String() string {
	return string(t)
}

const (
	// ChannelEventTopicChanged チャンネルイベント トピック変更
	//
	// 	userId 変更者UUID
	// 	before 変更前トピック
	// 	after  変更後トピック
	ChannelEventTopicChanged = ChannelEventType("TopicChanged")
	// ChannelEventSubscribersChanged チャンネルイベント 購読者変更
	//
	// 	userId 変更者UUID
	// 	on     オンにしたユーザーのUUIDの配列
	// 	off    オフにしたユーザーのUUIDの配列
	ChannelEventSubscribersChanged = ChannelEventType("SubscribersChanged")
	// ChannelEventPinAdded チャンネルイベント ピン追加
	//
	// 	userId    変更者UUID
	// 	messageId メッセージUUID
	ChannelEventPinAdded = ChannelEventType("PinAdded")
	// ChannelEventPinRemoved チャンネルイベント ピン削除
	//
	// 	userId    変更者UUID
	// 	messageId メッセージUUID
	ChannelEventPinRemoved = ChannelEventType("PinRemoved")
	// ChannelEventNameChanged チャンネルイベント 名前変更
	//
	// 	userId 変更者UUID
	// 	before 変更前名前
	// 	after  変更後名前
	ChannelEventNameChanged = ChannelEventType("NameChanged")
	// ChannelEventParentChanged チャンネルイベント 親チャンネル変更
	//
	// 	userId 変更者UUID
	// 	before 変更前親チャンネルUUID
	// 	after  変更後親チャンネルUUID
	ChannelEventParentChanged = ChannelEventType("ParentChanged")
	// ChannelEventVisibilityChanged チャンネルイベント 可視状態変更
	//
	// 	userId     変更者UUID
	// 	visibility 可視状態
	ChannelEventVisibilityChanged = ChannelEventType("VisibilityChanged")
	// ChannelEventForcedNotificationChanged チャンネルイベント 強制通知変更
	//
	// 	userId 変更者UUID
	// 	force  強制状態
	ChannelEventForcedNotificationChanged = ChannelEventType("ForcedNotificationChanged")
	// ChannelEventChildCreated チャンネルイベント 子チャンネル作成
	//
	// 	userId    作成者UUID
	// 	channelId チャンネルUUID
	ChannelEventChildCreated = ChannelEventType("ChildCreated")
)

// ChannelEventDetail チャンネルイベント詳細
type ChannelEventDetail map[string]interface{}

// Value database/sql/driver.Valuer 実装
func (ced ChannelEventDetail) Value() (driver.Value, error) {
	return json.Marshal(ced)
}

// Scan database/sql.Scanner 実装
func (ced *ChannelEventDetail) Scan(src interface{}) error {
	*ced = ChannelEventDetail{}
	switch s := src.(type) {
	case nil:
		return nil
	case string:
		return json.Unmarshal([]byte(s), ced)
	case []byte:
		return json.Unmarshal(s, ced)
	default:
		return errors.New("failed to scan ChannelEventDetail")
	}
}

// ChannelEvent チャンネルイベント
type ChannelEvent struct {
	EventID   uuid.UUID          `gorm:"type:char(36);not null;primary_key"`
	ChannelID uuid.UUID          `gorm:"type:char(36);not null"`
	EventType ChannelEventType   `gorm:"type:varchar(30);not null;"`
	Detail    ChannelEventDetail `sql:"type:TEXT COLLATE utf8mb4_bin NOT NULL"`
	DateTime  time.Time          `gorm:"precision:6"`
}

// TableName テーブル名
func (*ChannelEvent) TableName() string {
	return "channel_events"
}
