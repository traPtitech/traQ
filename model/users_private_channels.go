package model

import "fmt"

// UsersPrivateChannel : UsersPrivateChannelsの構造体
type UsersPrivateChannel struct {
	UserID    string `xorm:"char(36) pk"`
	ChannelID string `xorm:"char(36) pk"`
}

// TableName : テーブル名を指定するメソッド
func (usersPrivateChannel *UsersPrivateChannel) TableName() string {
	return "users_private_channels"
}

// Create : データベースへ反映
func (usersPrivateChannel *UsersPrivateChannel) Create() error {
	if usersPrivateChannel.ChannelID == "" {
		return fmt.Errorf("ChannelId is empty")
	}

	if usersPrivateChannel.UserID == "" {
		return fmt.Errorf("UserId is empty")
	}

	if _, err := db.Insert(usersPrivateChannel); err != nil {
		return fmt.Errorf("Failed to create user_private_channel: %v", err)
	}

	return nil
}
