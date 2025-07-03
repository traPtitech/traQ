package model

import (
	"database/sql/driver"
	"errors"
	"time"

	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

const (
	// DirectMessageChannelRootID ダイレクトメッセージチャンネルの親チャンネルID
	DirectMessageChannelRootID = "aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa"
)

var dmChannelRootUUID = uuid.Must(uuid.FromString(DirectMessageChannelRootID))

// Channel チャンネルの構造体
type Channel struct {
	ID        uuid.UUID      `gorm:"type:char(36);not null;primaryKey;index:idx_channel_channels_id_is_public_is_forced,priority:1"`
	Name      string         `gorm:"type:varchar(20);not null;uniqueIndex:name_parent"`
	ParentID  uuid.UUID      `gorm:"type:char(36);not null;uniqueIndex:name_parent"`
	Topic     string         `gorm:"type:TEXT COLLATE utf8mb4_bin NOT NULL"`
	IsForced  bool           `gorm:"type:boolean;not null;default:false;index:idx_channel_channels_id_is_public_is_forced,priority:3"`
	IsPublic  bool           `gorm:"type:boolean;not null;default:false;index:idx_channel_channels_id_is_public_is_forced,priority:2"`
	IsVisible bool           `gorm:"type:boolean;not null;default:false"`
	IsThread  bool			 `gorm:"type:boolean;not null;default:false"`
	CreatorID uuid.UUID      `gorm:"type:char(36);not null"`
	UpdaterID uuid.UUID      `gorm:"type:char(36);not null"`
	CreatedAt time.Time      `gorm:"precision:6"`
	UpdatedAt time.Time      `gorm:"precision:6"`
	DeletedAt gorm.DeletedAt `gorm:"precision:6"`

	ChildrenID []uuid.UUID `gorm:"-"`
}

// TableName テーブル名を指定するメソッド
func (ch *Channel) TableName() string {
	return "channels"
}

// IsDMChannel ダイレクトメッセージ用チャンネルかどうかを返します
func (ch *Channel) IsDMChannel() bool {
	return ch.ParentID == dmChannelRootUUID
}

// IsArchived アーカイブされているチャンネルかどうか
func (ch *Channel) IsArchived() bool {
	return !ch.IsVisible
}

// UsersPrivateChannel UsersPrivateChannelsの構造体
type UsersPrivateChannel struct {
	UserID    uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
	ChannelID uuid.UUID `gorm:"type:char(36);not null;primaryKey"`

	User    User    `gorm:"constraint:users_private_channels_user_id_users_id_foreign,OnUpdate:CASCADE,OnDelete:CASCADE"`
	Channel Channel `gorm:"constraint:users_private_channels_channel_id_channels_id_foreign,OnUpdate:CASCADE,OnDelete:CASCADE"`
}

// TableName テーブル名を指定するメソッド
func (upc *UsersPrivateChannel) TableName() string {
	return "users_private_channels"
}

// ChannelSubscribeLevel チャンネル購読レベル
type ChannelSubscribeLevel int

const (
	// ChannelSubscribeLevelNone レベル：無し
	ChannelSubscribeLevelNone ChannelSubscribeLevel = iota
	// ChannelSubscribeLevelMark レベル：未読管理のみ
	ChannelSubscribeLevelMark
	// ChannelSubscribeLevelMarkAndNotify レベル：未読管理＋通知
	ChannelSubscribeLevelMarkAndNotify
	// ChannelSubscribeLevelMarkAndNotify
)

func (v ChannelSubscribeLevel) Int() int {
	return int(v)
}

// UserSubscribeChannel ユーザー・通知チャンネル対構造体
type UserSubscribeChannel struct {
	UserID    uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
	ChannelID uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
	Mark      bool      `gorm:"type:boolean;not null;default:false"`
	Notify    bool      `gorm:"type:boolean;not null;default:false"`

	User    User    `gorm:"constraint:users_subscribe_channels_user_id_users_id_foreign,OnUpdate:CASCADE,OnDelete:CASCADE"`
	Channel Channel `gorm:"constraint:users_subscribe_channels_channel_id_channels_id_foreign,OnUpdate:CASCADE,OnDelete:CASCADE"`
}

// TableName UserNotifiedChannel構造体のテーブル名
func (*UserSubscribeChannel) TableName() string {
	return "users_subscribe_channels"
}

// GetLevel 購読レベルを返します
func (usc *UserSubscribeChannel) GetLevel() ChannelSubscribeLevel {
	switch {
	case usc.Notify:
		return ChannelSubscribeLevelMarkAndNotify
	case usc.Mark:
		return ChannelSubscribeLevelMark
	default:
		return ChannelSubscribeLevelNone
	}
}

// DMChannelMapping ダイレクトメッセージチャンネルとユーザーのマッピング
type DMChannelMapping struct {
	ChannelID uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
	User1     uuid.UUID `gorm:"type:char(36);not null;uniqueIndex:user1_user2"`
	User2     uuid.UUID `gorm:"type:char(36);not null;uniqueIndex:user1_user2"`

	Channel *Channel `gorm:"constraint:dm_channel_mappings_channel_id_channels_id_foreign,OnUpdate:CASCADE,OnDelete:CASCADE"`
	U1      *User    `gorm:"constraint:dm_channel_mappings_user_one_users_id_foreign,OnUpdate:CASCADE,OnDelete:CASCADE;foreignKey:User1"` // NOTE: constraint nameに数字が入るとgormにinvalid name扱いされてしまう
	U2      *User    `gorm:"constraint:dm_channel_mappings_user_two_users_id_foreign,OnUpdate:CASCADE,OnDelete:CASCADE;foreignKey:User2"`
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
	return json.MarshalToString(ced)
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
	EventID   uuid.UUID          `gorm:"type:char(36);not null;primaryKey" json:"-"`
	ChannelID uuid.UUID          `gorm:"type:char(36);not null;index:idx_channel_events_channel_id_date_time,priority:1;index:idx_channel_events_channel_id_event_type_date_time,priority:1" json:"-"`
	EventType ChannelEventType   `gorm:"type:varchar(30);not null;index:idx_channel_events_channel_id_event_type_date_time,priority:2" json:"type"`
	Detail    ChannelEventDetail `gorm:"type:TEXT COLLATE utf8mb4_bin NOT NULL" json:"detail"`
	DateTime  time.Time          `gorm:"precision:6;index:idx_channel_events_channel_id_date_time,priority:2;index:idx_channel_events_channel_id_event_type_date_time,priority:3" json:"datetime"`

	Channel *Channel `gorm:"constraint:channel_events_channel_id_channels_id_foreign,OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
}

// TableName テーブル名
func (*ChannelEvent) TableName() string {
	return "channel_events"
}



type UserSubscribeThread struct {
	UserID    uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
	ChannelID uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
	Mark      bool      `gorm:"type:boolean;not null;default:false"`
	Notify    bool      `gorm:"type:boolean;not null;default:false"`

	User    User    `gorm:"constraint:users_subscribe_channels_user_id_users_id_foreign,OnUpdate:CASCADE,OnDelete:CASCADE"`
	Channel Channel `gorm:"constraint:users_subscribe_channels_channel_id_channels_id_foreign,OnUpdate:CASCADE,OnDelete:CASCADE"`
}

// TableName UserNotifiedChannel構造体のテーブル名
func (*UserSubscribeThread) TableName() string {
	return "users_subscribe_threads"
}

// GetLevel 購読レベルを返します
func (ust *UserSubscribeThread) GetLevel() ChannelSubscribeLevel {
	switch {
	case ust.Notify:
		return ChannelSubscribeLevelMarkAndNotify
	case ust.Mark:
		return ChannelSubscribeLevelMark
	default:
		return ChannelSubscribeLevelNone
	}
}