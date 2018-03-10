package model

import (
	"errors"
	"time"
)

var (
	// ErrInvisibleChannelInvalidField channelIDかuserIDが未入力
	ErrInvisibleChannelInvalidField = errors.New("invalid field")
)

// UserInvisibleChannel users_invisible_channelsの構造体
type UserInvisibleChannel struct {
	UserID    string    `xorm:"char(36) pk not null"`
	ChannelID string    `xorm:"char(36) pk not null"`
	CreatedAt time.Time `xorm:"created not null"`
}

// TableName テーブルの名前を指定する
func (*UserInvisibleChannel) TableName() string {
	return "users_invisible_channels"
}

// Create DBに登録する
func (i *UserInvisibleChannel) Create() error {
	if err := validateUserInvisibleChannel(i); err != nil {
		return err
	}

	if _, err := db.InsertOne(i); err != nil {
		return err
	}

	return nil
}

// Exists DBに登録されているかを確認する
func (i *UserInvisibleChannel) Exists() (bool, error) {
	if err := validateUserInvisibleChannel(i); err != nil {
		return false, err
	}

	return db.Get(i)
}

// Delete DBから削除する
func (i *UserInvisibleChannel) Delete() error {
	if err := validateUserInvisibleChannel(i); err != nil {
		return err
	}

	if _, err := db.Delete(i); err != nil {
		return err
	}
	return nil
}

// GetInvisibleChannelsByID 指定されたユーザーから見えないチャンネルのリストを取得する
func GetInvisibleChannelsByID(userID string) ([]string, error) {
	var channelIDs []string
	// FIXME: ここのクエリが不完全
	err := db.Table("channels").Join("LEFT", []string{"users_private_channels", "p"}, "p.channel_id = channels.id").
		Join("LEFT", []string{"users_invisible_channels", "i"}, "i.channel_id = channels.id").
		Where("is_deleted = true OR i.user_id = ? OR is_public = false", userID). // 自分のprivatechannelも取得してしまう
		Cols("id").Find(&channelIDs)
	if err != nil {
		return nil, err
	}
	return channelIDs, nil
}

// GetVisibleChannelsByID 指定されたユーザーから見えるチャンネルのリストを取得する
func GetVisibleChannelsByID(userID string) ([]string, error) {
	var channelIDs []string
	err := db.Table("channels").Join("LEFT", []string{"users_private_channels", "p"}, "p.channel_id = channels.id").
		Join("LEFT", []string{"users_invisible_channels", "i"}, "i.channel_id = channels.id").
		Where("(is_public = true OR p.user_id = ?) AND is_deleted = false AND i.user_id IS NULL", userID).
		Cols("id").Find(&channelIDs)
	if err != nil {
		return nil, err
	}
	return channelIDs, nil
}

// 外部キー制約の項目が入力されているかどうかを判定する
func validateUserInvisibleChannel(i *UserInvisibleChannel) error {
	if i.UserID == "" {
		return ErrInvisibleChannelInvalidField
	}
	if i.ChannelID == "" {
		return ErrInvisibleChannelInvalidField
	}

	return nil
}
