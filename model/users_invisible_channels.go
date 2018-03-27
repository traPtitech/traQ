package model

import (
	"github.com/traPtitech/traQ/utils/validator"
	"time"
)

// UserInvisibleChannel users_invisible_channelsの構造体
type UserInvisibleChannel struct {
	UserID    string    `xorm:"char(36) pk not null" validate:"uuid,required"`
	ChannelID string    `xorm:"char(36) pk not null" validate:"uuid,required"`
	CreatedAt time.Time `xorm:"created not null"`
}

// TableName テーブルの名前を指定する
func (*UserInvisibleChannel) TableName() string {
	return "users_invisible_channels"
}

// Validate 構造体を検証します
func (i *UserInvisibleChannel) Validate() error {
	return validator.ValidateStruct(i)
}

// Create DBに登録する
func (i *UserInvisibleChannel) Create() (err error) {
	if err = i.Validate(); err != nil {
		return err
	}

	_, err = db.InsertOne(i)
	return
}

// Exists DBに登録されているかを確認する
func (i *UserInvisibleChannel) Exists() (bool, error) {
	if err := i.Validate(); err != nil {
		return false, err
	}

	return db.Get(i)
}

// Delete DBから削除する
func (i *UserInvisibleChannel) Delete() (err error) {
	if err = i.Validate(); err != nil {
		return err
	}

	_, err = db.Delete(i)
	return
}

// GetInvisibleChannelsByID 指定されたユーザーから見えないチャンネルのリストを取得する
func GetInvisibleChannelsByID(userID string) (channelIDs []string, err error) {
	// FIXME: ここのクエリが不完全
	err = db.Table("channels").Join("LEFT", []string{"users_private_channels", "p"}, "p.user_id = ? AND p.channel_id = channels.id", userID).
		Join("LEFT", []string{"users_invisible_channels", "i"}, "i.user_id = ? AND i.channel_id = channels.id", userID).
		Where("i.channel_id IS NOT NULL OR is_visible = false OR (p.channel_id IS NULL AND is_public = false) OR is_deleted = true").
		Cols("id").Find(&channelIDs)
	return
}

// GetVisibleChannelsByID 指定されたユーザーから見えるチャンネルのリストを取得する
func GetVisibleChannelsByID(userID string) (channelIDs []string, err error) {
	err = db.Table("channels").Join("LEFT", []string{"users_private_channels", "p"}, "p.user_id = ? AND p.channel_id = channels.id", userID).
		Join("LEFT", []string{"users_invisible_channels", "i"}, "i.user_id = ? AND i.channel_id = channels.id", userID).
		Where("(is_public = true OR p.channel_id IS NOT NULL) AND NOT (is_deleted = true OR is_visible = false OR i.channel_id IS NOT NULL)").
		Cols("id").Find(&channelIDs)
	return
}
