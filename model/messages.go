package model

import (
	"fmt"
)

//Messages struct: データベースに格納するmessageの構造体
type Messages struct {
	Id        string `xorm:"char(36) pk"`
	UserId    string `xorm:"char(36) not null"`
	ChannelId string `xorm:"char(36)"`
	Text      string `xorm:"text not null"`
	IsShared  bool   `xorm:"bool not null"`
	IsDeleted bool   `xorm:"bool not null"`
	CreatedAt string `xorm:"created not null"`
	UpdaterId string `xorm:"char(36) not null"`
	UpdatedAt string `xorm:"updated not null"`
}

// Create method inserts message object to database.
func (message *Messages) Create() error {
	if message.UserId == "" {
		return fmt.Errorf("UserId is empty")
	}

	if message.Text == "" {
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

// Update method:受け取ったメッセージIDの本文を変更します
func (message *Messages) Update() error {
	_, err := db.ID(message.Id).UseBool().Update(message)
	if err != nil {
		return fmt.Errorf("Failed to update this message: %v", err)
	}
	return nil
}

// GetMessagesFromChannel :指定されたチャンネルのメッセージを取得します
func GetMessagesFromChannel(channelId string) ([]*Messages, error) {
	var messageList []*Messages
	err := db.Where("channel_id = ?", channelId).Find(messageList)
	if err != nil {
		return nil, fmt.Errorf("Failed to find messages: %v", err)
	}

	return messageList, nil
}

// GetMessage :messageIdで指定されたメッセージを取得します
func GetMessage(messageId string) (*Messages, error) {
	var message *Messages
	has, err := db.ID(messageId).Get(message)

	if err != nil {
		return nil, fmt.Errorf("Failed to find message: %v", err)
	}
	if has == false {
		return nil, fmt.Errorf("This messageId is wrong")
	}

	return message, nil
}

// DeleteMessage :messageIdで指定されたメッセージのIsdeleteをtrueにします
func DeleteMessage(messageId string) error {

	message, err := GetMessage(messageId)
	if err != nil {
		return fmt.Errorf("Failed to get message: %v", err)
	}

	message.IsDeleted = true

	if _, err := db.ID(messageId).UseBool().Update(message); err != nil {
		return fmt.Errorf("Failed to update message: %v", err)
	}

	return nil
}
