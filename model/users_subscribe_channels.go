package model

import (
	"fmt"
	"github.com/traPtitech/traQ/utils/validator"
	"time"

	"github.com/satori/go.uuid"
)

// UserSubscribeChannel ユーザー・通知チャンネル対構造体
type UserSubscribeChannel struct {
	UserID    string    `xorm:"char(36) pk not null" validate:"uuid,required"`
	ChannelID string    `xorm:"char(36) pk not null" validate:"uuid,required"`
	CreatedAt time.Time `xorm:"created not null"`
}

// TableName UserNotifiedChannel構造体のテーブル名
func (*UserSubscribeChannel) TableName() string {
	return "users_subscribe_channels"
}

// Validate 構造体を検証します
func (s *UserSubscribeChannel) Validate() error {
	return validator.ValidateStruct(s)
}

// Create DBに登録
func (s *UserSubscribeChannel) Create() (err error) {
	if err = s.Validate(); err != nil {
		return err
	}

	_, err = db.Insert(s)
	return
}

// Delete DBから削除
func (s *UserSubscribeChannel) Delete() (err error) {
	if err = s.Validate(); err != nil {
		return err
	}

	_, err = db.Delete(s)
	return
}

// GetSubscribingUser 指定したチャンネルの通知をつけているユーザーを取得
func GetSubscribingUser(channelID uuid.UUID) ([]uuid.UUID, error) {
	var arr []string
	if err := db.Table(&UserSubscribeChannel{}).Where("channel_id = ?", channelID.String()).Cols("user_id").Find(&arr); err != nil {
		return nil, fmt.Errorf("failed to get user_subscribe_channel: %v", err)
	}

	result := make([]uuid.UUID, len(arr))
	for i, v := range arr {
		result[i] = uuid.FromStringOrNil(v)
	}

	return result, nil
}
