package model

import (
	"github.com/satori/go.uuid"
)

// UsersPrivateChannel UsersPrivateChannelsの構造体
type UsersPrivateChannel struct {
	UserID    uuid.UUID `gorm:"type:char(36);primary_key"`
	ChannelID uuid.UUID `gorm:"type:char(36);primary_key"`
}

// TableName テーブル名を指定するメソッド
func (upc *UsersPrivateChannel) TableName() string {
	return "users_private_channels"
}

// AddPrivateChannelMember プライベートチャンネルにメンバーを追加します
func AddPrivateChannelMember(channelID, userID uuid.UUID) error {
	if err := db.Create(&UsersPrivateChannel{UserID: userID, ChannelID: channelID}).Error; err != nil {
		if isMySQLDuplicatedRecordErr(err) {
			return nil
		}
		return err
	}
	return nil
}

// GetPrivateChannelMembers プライベートチャンネルのメンバーの配列を取得する
func GetPrivateChannelMembers(channelID uuid.UUID) (member []uuid.UUID, err error) {
	member = make([]uuid.UUID, 0)
	err = db.Model(UsersPrivateChannel{}).Where(&UsersPrivateChannel{ChannelID: channelID}).Pluck("user_id", &member).Error
	return
}

// IsUserPrivateChannelMember ユーザーがプライベートチャンネルのメンバーかどうかを確認します
func IsUserPrivateChannelMember(channelID, userID uuid.UUID) (bool, error) {
	c := 0
	err := db.Model(UsersPrivateChannel{}).Where(&UsersPrivateChannel{ChannelID: channelID, UserID: userID}).Count(&c).Error
	if err != nil {
		return false, err
	}
	return c > 0, nil
}
