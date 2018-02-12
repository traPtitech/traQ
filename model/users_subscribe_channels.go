package model

import (
	"fmt"
	"github.com/satori/go.uuid"
	"time"
)

// ユーザー・通知チャンネル対構造体
type UserSubscribeChannel struct {
	UserId    string    `xorm:"char(36) pk not null"`
	ChannelId string    `xorm:"char(36) pk not null"`
	CreatedAt time.Time `xorm:"created not null"`
}

// UserNotifiedChannel構造体のテーブル名
func (*UserSubscribeChannel) TableName() string {
	return "users_notified_channels"
}

// DBに登録
func (s *UserSubscribeChannel) Create() error {
	if s.UserId == "" {
		return fmt.Errorf("UserId is empty")
	}
	if s.ChannelId == "" {
		return fmt.Errorf("ChannelId is empty")
	}

	if _, err := db.Insert(s); err != nil {
		return fmt.Errorf("failed to create user_notified_channel: %v", err)
	}

	return nil
}

// DBから削除
func (s *UserSubscribeChannel) Delete() error {
	if s.UserId == "" {
		return fmt.Errorf("UserId is empty")
	}
	if s.ChannelId == "" {
		return fmt.Errorf("ChannelId is empty")
	}

	if _, err := db.Delete(s); err != nil {
		return fmt.Errorf("failed to delete user_notified_channel: %v", err)
	}

	return nil
}

// 指定したチャンネルの通知をつけているユーザーを取得
func GetSubscribingUser(channelId uuid.UUID) ([]uuid.UUID, error) {
	var arr []string
	if err := db.Table(UserSubscribeChannel{}.TableName()).Where("channel_id = ?", channelId.String()).Cols("user_id").Find(&arr); err != nil {
		return nil, fmt.Errorf("failed to get user_subscribe_channel: %v", err)
	}

	result := make([]uuid.UUID, len(arr))
	for i, v := range arr {
		result[i] = uuid.FromStringOrNil(v)
	}

	return result, nil
}
