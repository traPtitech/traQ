package model

import (
	"fmt"
)

type Pin struct {
	ChannelID string `xorm:"char(36) pk"`
	MessageID string `xorm:"char(36) pk"`
	UserID    string `xorm:"char(36) not null"`
	CreateAt  string `xorm:"created not null"`
}

func (pin *Pin) Tablename() string {
	return "pins"
}

func (pin *Pin) Create() error {
	if pin.UserID == "" {
		return fmt.Errorf("UserID is empty")
	}
	if pin.ChannelID == "" {
		return fmt.Errorf("ChannelID is empty")
	}
	if pin.MessageID == "" {
		return fmt.Errorf("MessageID is empty")
	}

	if _, err := db.Insert(pin); err != nil {
		return fmt.Errorf("Failed to create pin object: %v", err)
	}

	return nil
}

func GetPinMesssages(channelID string) ([]*Message, error) {
	if channelID == "" {
		return nil, fmt.Errorf("ChannelId is empty")
	}
	var messages []*Message

	err := db.Table("pins").Join("LEFT", "messages", "pins.message_id = messages.id").Where("pins.channel_id = ? AND is_deleted = false", channelID).Find(&messages)

	if err != nil {
		return nil, fmt.Errorf("Failed to find pined messages: %v", err)
	}

	return messages, nil
}

func (pin *Pin) DeletePin() error {
	if pin.ChannelID == "" {
		return fmt.Errorf("ChannelID is empty")
	}

	if pin.MessageID == "" {
		return fmt.Errorf("MessageID is empty")
	}

	if _, err := db.Delete(pin); err != nil {
		return fmt.Errorf("Fail to delete pin: %v", err)
	}
	return nil
}
