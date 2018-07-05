package model

import (
	"github.com/satori/go.uuid"
	"time"
)

// UserSubscribeChannel ユーザー・通知チャンネル対構造体
type UserSubscribeChannel struct {
	UserID    string    `gorm:"type:char(36);primary_key"`
	ChannelID string    `gorm:"type:char(36);primary_key"`
	CreatedAt time.Time `gorm:"precision:6"`
}

// TableName UserNotifiedChannel構造体のテーブル名
func (*UserSubscribeChannel) TableName() string {
	return "users_subscribe_channels"
}

// SubscribeChannel 指定したチャンネルを購読します
func SubscribeChannel(userID, channelID uuid.UUID) error {
	return db.Create(&UserSubscribeChannel{UserID: userID.String(), ChannelID: channelID.String()}).Error
}

// UnsubscribeChannel 指定したチャンネルの購読を解除します
func UnsubscribeChannel(userID, channelID uuid.UUID) error {
	return db.Where(UserSubscribeChannel{UserID: userID.String(), ChannelID: channelID.String()}).Delete(UserSubscribeChannel{}).Error
}

// GetSubscribingUser 指定したチャンネルを購読しているユーザーを取得
func GetSubscribingUser(channelID uuid.UUID) ([]uuid.UUID, error) {
	var arr []string
	err := db.Model(UserSubscribeChannel{}).Where(UserSubscribeChannel{ChannelID: channelID.String()}).Pluck("user_id", &arr).Error
	if err != nil {
		return nil, err
	}
	return convertStringSliceToUUIDSlice(arr), nil
}

// GetSubscribedChannels ユーザーが購読しているチャンネルを取得する
func GetSubscribedChannels(userID uuid.UUID) ([]uuid.UUID, error) {
	var arr []string
	err := db.Model(UserSubscribeChannel{}).Where(UserSubscribeChannel{UserID: userID.String()}).Pluck("channel_id", &arr).Error
	if err != nil {
		return nil, err
	}
	return convertStringSliceToUUIDSlice(arr), nil
}
