package model

import (
	"github.com/traPtitech/traQ/utils/validator"
)

// UsersPrivateChannel : UsersPrivateChannelsの構造体
type UsersPrivateChannel struct {
	UserID    string `xorm:"char(36) pk" validate:"uuid,required"`
	ChannelID string `xorm:"char(36) pk" validate:"uuid,required"`
}

// TableName : テーブル名を指定するメソッド
func (upc *UsersPrivateChannel) TableName() string {
	return "users_private_channels"
}

// Validate 構造体を検証します
func (upc *UsersPrivateChannel) Validate() error {
	return validator.ValidateStruct(upc)
}

// Create : データベースへ反映
func (upc *UsersPrivateChannel) Create() (err error) {
	if err = upc.Validate(); err != nil {
		return err
	}

	_, err = db.InsertOne(upc)
	return
}

// GetPrivateChannel ある二つのユーザー間のプライベートチャンネルが存在するかを調べる
func GetPrivateChannel(userID1, userID2 string) (*UsersPrivateChannel, error) {
	upc := &UsersPrivateChannel{}
	has, err := db.Table(upc).Join("LEFT", []string{"users_private_channels", "p"}, "p.user_id = ? AND users_private_channels.user_id = ? AND p.channel_id = users_private_channels.channel_id", userID1, userID2).
		Get(upc)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, ErrNotFound
	}
	return upc, nil
}

// GetMembers DMのメンバーの配列を取得する
func GetMembers(channelID string) (member []string, err error) {
	err = db.Table("users_private_channels").Where("channel_id = ?", channelID).Cols("user_id").Find(&member)
	return
}
