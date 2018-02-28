package model

import (
	"fmt"
	"time"
)

//Pin ピン留めのレコード
type Pin struct {
	ID        string    `xorm:"char(36) pk"`
	ChannelID string    `xorm:"char(36) not null"`
	MessageID string    `xorm:"char(36) not null"`
	UserID    string    `xorm:"char(36) not null"`
	CreatedAt time.Time `xorm:"created not null"`
}

//TableName ピン留めテーブル名
func (pin *Pin) TableName() string {
	return "pins"
}

//Create ピン留めレコードを追加する
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

	pin.ID = CreateUUID()

	if _, err := db.Insert(pin); err != nil {
		return fmt.Errorf("Failed to create pin: %v", err)
	}

	return nil
}

//GetPin IDからピン留めを取得する
func GetPin(ID string) (*Pin, error) {
	if ID == "" {
		return nil, fmt.Errorf("ID is empty")
	}

	pin := &Pin{}
	if has, err := db.ID(ID).Get(pin); err != nil {
		return nil, fmt.Errorf("Failed to get pin: %v", err)
	} else if !has {
		return nil, nil
	}

	return pin, nil
}

//GetPinsByChannelID あるチャンネルのピン留めを全部取得する
func GetPinsByChannelID(channelID string) ([]*Pin, error) {
	if channelID == "" {
		return nil, fmt.Errorf("ChannelID is empty")
	}

	var pins []*Pin
	if err := db.Where("channel_id = ?", channelID).Find(&pins); err != nil {
		return nil, fmt.Errorf("Failed to find pins: %v", err)
	}

	return pins, nil
}

//Delete ピン留めレコードを削除する
func (pin *Pin) Delete() error {
	if pin.ID == "" {
		return fmt.Errorf("ID is empty")
	}

	if _, err := db.Delete(pin); err != nil {
		return fmt.Errorf("Failed to delete pin: %v", err)
	}
	return nil
}
