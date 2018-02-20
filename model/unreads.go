package model

import (
	"fmt"
)

//Unread 未読レコード
type Unread struct {
	UserID    string `xrom:"char(36) not null pk"`
	MessageID string `xorm:"char(36) not null pk"`
}

//TableName テーブル名
func (unread *Unread) TableName() string {
	return "unreads"
}

//Create レコード作成
func (unread *Unread) Create() error {
	if unread.UserID == "" {
		return fmt.Errorf("userID is empty")
	}

	if unread.MessageID == "" {
		return fmt.Errorf("messageID is empty")
	}

	if _, err := db.Insert(unread); err != nil {
		return fmt.Errorf("Failed to create unread: %v", err)
	}
	return nil
}

//Delete レコード削除
func (unread *Unread) Delete() error {
	if unread.UserID == "" {
		return fmt.Errorf("userID is empty")
	}

	if unread.MessageID == "" {
		return fmt.Errorf("messageID is empty")
	}

	if _, err := db.Delete(unread); err != nil {
		return fmt.Errorf("Failed to delete unread: %v", err)
	}
	return nil
}

//GetUnreadsByUserID あるユーザーの未読レコードをすべて取得
func GetUnreadsByUserID(userID string) ([]*Unread, error) {
	if userID == "" {
		return nil, fmt.Errorf("userID is empty")
	}

	var unreads []*Unread
	if err := db.Where("user_id = ?", userID).Find(&unreads); err != nil {
		return nil, fmt.Errorf("Failed to find unreads: %v", err)
	}
	return unreads, nil
}

//DeleteUnreadsByMessageID 指定したメッセージIDの未読レコードを全て削除
func DeleteUnreadsByMessageID(messageID string) error {
	if messageID == "" {
		return fmt.Errorf("messageID is empty")
	}

	if _, err := db.Delete(&Unread{MessageID: messageID}); err != nil {
		return err
	}
	return nil
}
