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
func GetPrivateChannel(userID1, userID2 string) (string, error) {
	// *string型の変数でchannelIDのみをGetしようとするとエラーを吐く
	upc := &UsersPrivateChannel{}
	if userID1 == userID2 {
		// 自分宛てのDMのときの処理
		has, err := db.SQL("SELECT channel_id FROM users_private_channels GROUP BY channel_id HAVING COUNT(*) = 1 AND GROUP_CONCAT(user_id) = ?", userID1).Get(upc)
		if err != nil {
			return "", err
		}
		if !has {
			return "", ErrNotFound
		}
	} else {
		// HACK: よりよいクエリ文が見つかったら変える
		has, err := db.SQL("SELECT u.channel_id FROM users_private_channels AS u INNER JOIN (SELECT channel_id FROM users_private_channels GROUP BY channel_id HAVING COUNT(*) = 2) AS ex ON ex.channel_id = u.channel_id AND u.user_id IN(?, ?) GROUP BY channel_id HAVING COUNT(*) = 2", userID1, userID2).Get(upc)
		if err != nil {
			return "", err
		}
		if !has {
			return "", ErrNotFound
		}
	}
	return upc.ChannelID, nil
}

// GetPrivateChannelMembers DMのメンバーの配列を取得する
func GetPrivateChannelMembers(channelID string) (member []string, err error) {
	err = db.Table("users_private_channels").Where("channel_id = ?", channelID).Cols("user_id").Find(&member)
	return
}
