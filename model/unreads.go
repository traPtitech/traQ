package model

import (
	"fmt"
)

type Unread struct {
	UserID    string `xrom:"char(36) not null pk"`
	MessageID string `xorm:"char(36) not null pk"`
}

func (unread *Unread) TableName() string {
	return "unreads"
}

func (unread *Unread) Create() error {
	if unread.UserID == "" {
		return fmt.Errorf("UserID is empty.")
	}

	if unread.MessageID == "" {
		return fmt.Errorf("MessageID is empty.")
	}

	if _, err := db.Insert(unread); err != nil {
		return fmt.Errorf("Failed to create unread: %v", err)
	}
	return nil
}

func (unread *Unread) Delete() error {
	if unread.UserID == "" {
		return fmt.Errorf("UserID is empty.")
	}

	if unread.MessageID == "" {
		return fmt.Errorf("MessageID is empty.")
	}

	if _, err := db.Delete(unread); err != nil {
		return fmt.Errorf("Failed to delete unread: %v", err)
	}
	return nil
}

func GetUnreadsByUserID(userID string) ([]*Unread, error) {
	var unreads []*Unread
	if err := db.Where("user_id = ?", userID).Find(&unreads); err != nil {
		return nil, fmt.Errorf("Failed to find unreads: %v", err)
	}
	return unreads, nil
}
