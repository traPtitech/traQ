package model

import (
	"fmt"
	"time"
)

// Clip clipの構造体
type Clip struct {
	UserID    string    `xorm:"char(36) pk"`
	MessageID string    `xorm:"char(36) pk"`
	CreatedAt time.Time `xorm:"created not null"`
}

// TableName dbの名前を指定する
func (clip *Clip) TableName() string {
	return "clips"
}

// Create clipをDBに入れる
func (clip *Clip) Create() error {
	if clip.UserID == "" {
		return fmt.Errorf("UserID is empty")
	}

	if clip.MessageID == "" {
		return fmt.Errorf("MessageID is empty")
	}

	if _, err := db.Insert(clip); err != nil {
		return fmt.Errorf("Failed to create message object: %v", err)
	}
	return nil
}

// GetClippedMessages userIDがクリップしているメッセージの一覧を取得する
func GetClippedMessages(userID string) ([]*Message, error) {
	if userID == "" {
		return nil, fmt.Errorf("UserID is empty")
	}

	var messages []*Message
	err := db.Table("clips").Join("LEFT", "messages", "clips.message_id = messages.id").Where("clips.user_id = ? AND is_deleted = false", userID).Find(&messages)

	if err != nil {
		return nil, fmt.Errorf("Failed to find cliped messages: %v", err)
	}

	return messages, nil
}

// Delete clipを削除する
func (clip *Clip) Delete() error {
	if clip.UserID == "" {
		return fmt.Errorf("UserID is empty")
	}

	if clip.MessageID == "" {
		return fmt.Errorf("MessageID is empty")
	}

	if _, err := db.Delete(clip); err != nil {
		return fmt.Errorf("Failed to delete clip: %v", err)
	}

	return nil
}
