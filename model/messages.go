package model

import (
	"fmt"
)

//Messages struct: データベースに格納するmessageの構造体
type Messages struct {
	ID        string `xorm:"char(36) pk"`
	UserID    string `xorm:"char(36) not null"`
	ChannelID string `xorm:"char(36)"`
	Text      string `xorm:"text not null"`
	IsShared  bool   `xorm:"bool not null"`
	IsDeleted bool   `xorm:"bool not null"`
	CreatedAt string `xorm:"created not null"`
	UpdaterID string `xorm:"char(36) not null"`
	UpdatedAt string `xorm:"updated not null"`
}

// Create method inserts message object to database.
func (message *Messages) Create() error {
	if message.UserID == "" {
		return fmt.Errorf("UserID is empty")
	}

	if message.Text == "" {
		return fmt.Errorf("Text is empty")
	}

	message.ID = CreateUUID()
	message.IsDeleted = false
	message.UpdaterID = message.UserID

	if _, err := db.Insert(message); err != nil {
		return fmt.Errorf("Failed to create message object: %v", err)
	}
	return nil
}

// Update method:メッセージの内容を変更します
func (message *Messages) Update() error {
	_, err := db.ID(message.ID).UseBool().Update(message)
	if err != nil {
		return fmt.Errorf("Failed to update this message: %v", err)
	}
	return nil
}

// GetMessagesFromChannel :指定されたチャンネルのメッセージを取得します
func GetMessagesFromChannel(channelID string) ([]*Messages, error) {
	var messageList []*Messages
	err := db.Where("channel_id = ?", channelID).Asc("created_at").Find(&messageList)
	if err != nil {
		return nil, fmt.Errorf("Failed to find messages: %v", err)
	}

	return messageList, nil
}

// GetMessage :messageIDで指定されたメッセージを取得します
func GetMessage(messageID string) (*Messages, error) {
	var message = new(Messages)
	has, err := db.ID(messageID).Get(message)

	if err != nil {
		return nil, fmt.Errorf("Failed to find message: %v", err)
	}
	if has == false {
		return nil, fmt.Errorf("This messageID is wrong: messageID = %v", messageID)
	}

	return message, nil
}
