package model

import "fmt"

type UsersPrivateChannels struct {
	UserId    string `xorm:"user_id primary_key"`
	ChannelId string `xorm:"channel_id primary_key"`
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
