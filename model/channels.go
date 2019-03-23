package model

import (
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
	Topic     string     `gorm:"type:text;not null"`
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
