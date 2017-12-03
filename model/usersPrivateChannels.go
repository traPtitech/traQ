package model

import "fmt"

type UsersPrivateChannels struct {
	UserId    string `xorm:"char(36) pk"`
	ChannelId string `xorm:"char(36) pk"`
}

func (self *UsersPrivateChannels) Create() error {
	if self.ChannelId == "" {
		return fmt.Errorf("ChannelId is empty")
	}

	if self.UserId == "" {
		return fmt.Errorf("UserId is empty")
	}

	if _, err := db.Insert(self); err != nil {
		return fmt.Errorf("Failed to create user_private_channel: %v", err)
	}

	return nil
}
