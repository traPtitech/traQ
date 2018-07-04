package model

import (
	"github.com/jinzhu/gorm"
	"github.com/satori/go.uuid"
)

// UsersPrivateChannel UsersPrivateChannelsの構造体
type UsersPrivateChannel struct {
	UserID    string `gorm:"type:char(36);primary_key"`
	ChannelID string `gorm:"type:char(36);primary_key"`
}

// TableName テーブル名を指定するメソッド
func (upc *UsersPrivateChannel) TableName() string {
	return "users_private_channels"
}

// AddPrivateChannelMember プライベートチャンネルにメンバーを追加します
func AddPrivateChannelMember(channelID, userID uuid.UUID) error {
	upc := &UsersPrivateChannel{
		UserID:    userID.String(),
		ChannelID: channelID.String(),
	}
	return db.Create(&upc).Error
}

// GetPrivateChannel ある二つのユーザー間のプライベートチャンネルが存在するかを調べる
func GetPrivateChannel(userID1, userID2 string) (string, error) {
	// *string型の変数でchannelIDのみをGetしようとするとエラーを吐く
	channel := &UsersPrivateChannel{}
	if userID1 == userID2 {
		// 自分宛てのDMのときの処理
		err := db.
			Select("channel_id").
			Group("channel_id").
			Having("COUNT(*) = 1 AND GROUP_CONCAT(user_id) = ?", userID1).
			Take(&channel).
			Error
		if err != nil {
			if gorm.IsRecordNotFoundError(err) {
				return "", ErrNotFound
			}
			return "", err
		}
	} else {
		// HACK: よりよいクエリ文が見つかったら変える
		err := db.
			Raw("SELECT u.channel_id FROM users_private_channels AS u INNER JOIN (SELECT channel_id FROM users_private_channels GROUP BY channel_id HAVING COUNT(*) = 2) AS ex ON ex.channel_id = u.channel_id AND u.user_id IN(?, ?) GROUP BY channel_id HAVING COUNT(*) = 2", userID1, userID2).
			Take(&channel).
			Error
		if err != nil {
			if gorm.IsRecordNotFoundError(err) {
				return "", ErrNotFound
			}
			return "", err
		}
	}
	return channel.ChannelID, nil
}

// GetPrivateChannelMembers DMのメンバーの配列を取得する
func GetPrivateChannelMembers(channelID string) (member []string, err error) {
	err = db.Model(UserSubscribeChannel{}).Where(UserSubscribeChannel{ChannelID: channelID}).Pluck("user_id", &member).Error
	return
}

// IsUserPrivateChannelMember ユーザーがプライベートチャンネルのメンバーかどうかを確認します
func IsUserPrivateChannelMember(channelID, userID uuid.UUID) (bool, error) {
	c := &UserSubscribeChannel{}
	err := db.Where(UserSubscribeChannel{ChannelID: channelID.String(), UserID: userID.String()}).Take(c).Error
	if err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
