package model

import (
	"fmt"
)

type Messages struct {
	Id        string `xorm:char(36) pk`
	UserId    string `xorm:char(36) not null`
	ChannelId string `xorm:char(36)`
	text      string `xorm:text not null`
	IsShared  bool   `xorm:bool not null`
	IsDeleted bool   `xorm:bool not null`
	CreatedAt string `xorm:created not null`
	UpdaterId string `xorm:char(36) not null`
	UpdatedAt string `xorm:updated not null`
}

// Create inserts message object.
func (message *Messages) Create() error {
	if message.UserId == "" {
		return fmt.Errorf("UserId is empty")
	}

	if message.text == "" {
		return fmt.Errorf("Text is empty")
	}

	message.Id = CreateUUID()
	message.IsDeleted = false
	message.UpdaterId = message.UserId

	if _, err := db.Insert(message); err != nil {
		return fmt.Errorf("Failed to create message object: %v", err)
	}
	return nil
}

func (self *Messages) Update() error {
	return nil
}

func GetMessagesFromChannel(channelId string) ([]*Messages, error) {
	return nil, nil
}

func GetMessage(messageId string) (*Messages, error) {
	return nil, nil
}

// DeleteMessage deletes the message.
func DeleteMessage(messageId string) error {
	var message Messages
	has, err := db.ID(messageId).Get(&message)

	if err != nil {
		return fmt.Errorf("Failed to find message: %v", err)
	}
	if has == false {
		return fmt.Errorf("MessageId is wrong")
	}

	message.IsDeleted = true

	if _, err := db.ID(messageId).Update(&message); err != nil {
		return fmt.Errorf("Failed to update message: %v", err)
	}

	return nil
}
