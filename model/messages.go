package model

import (
	"fmt"
	"time"
)

//Message :データベースに格納するmessageの構造体
type Message struct {
	ID        string    `xorm:"char(36) pk"`
	UserID    string    `xorm:"char(36) not null"`
	ChannelID string    `xorm:"char(36)"`
	Text      string    `xorm:"text not null"`
	IsShared  bool      `xorm:"bool not null"`
	IsDeleted bool      `xorm:"bool not null"`
	CreatedAt time.Time `xorm:"created not null"`
	UpdaterID string    `xorm:"char(36) not null"`
	UpdatedAt time.Time `xorm:"updated not null"`
}

//TableName :DBの名前を指定するメソッド
func (message *Message) TableName() string {
	return "messages"
}

// Create method inserts message object to database.
func (message *Message) Create() error {
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
func (message *Message) Update() error {
	_, err := db.ID(message.ID).UseBool().Update(message)
	if err != nil {
		return fmt.Errorf("Failed to update this message: %v", err)
	}
	return nil
}

// GetMessagesFromChannel :指定されたチャンネルのメッセージを取得します
func GetMessagesFromChannel(channelID string, limit, offset int) ([]*Message, error) {
	var messageList []*Message
	err := db.Where("channel_id = ? AND is_deleted = false", channelID).Desc("created_at").Limit(limit, offset).Find(&messageList)
	if err != nil {
		return nil, fmt.Errorf("Failed to find messages: %v", err)
	}

	return messageList, nil
}

// GetMessage :messageIDで指定されたメッセージを取得します
func GetMessage(messageID string) (*Message, error) {
	var message = &Message{}
	has, err := db.ID(messageID).Get(message)

	if err != nil {
		return nil, fmt.Errorf("Failed to find message: %v", err)
	}
	if has == false {
		return nil, fmt.Errorf("This messageID is wrong: messageID = %v", messageID)
	}
	if message.IsDeleted {
		return nil, fmt.Errorf("this message has been deleted")
	}

	return message, nil
}
