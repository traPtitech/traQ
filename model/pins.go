package model

import (
	"fmt"
	"github.com/traPtitech/traQ/utils/validator"
	"time"
)

//Pin ピン留めのレコード
type Pin struct {
	ID        string    `xorm:"char(36) pk"       validate:"uuid,required"`
	ChannelID string    `xorm:"char(36) not null" validate:"uuid,required"`
	MessageID string    `xorm:"char(36) not null" validate:"uuid,required"`
	UserID    string    `xorm:"char(36) not null" validate:"uuid,required"`
	CreatedAt time.Time `xorm:"created not null"`
}

//TableName ピン留めテーブル名
func (pin *Pin) TableName() string {
	return "pins"
}

// Validate 構造体を検証します
func (pin *Pin) Validate() error {
	return validator.ValidateStruct(pin)
}

//Create ピン留めレコードを追加する
func (pin *Pin) Create() (err error) {
	pin.ID = CreateUUID()
	if err = pin.Validate(); err != nil {
		return
	}

	_, err = db.InsertOne(pin)
	return
}

// Exists pinが存在するかどうかを判定する
func (pin *Pin) Exists() (bool, error) {
	if pin.ID == "" {
		return false, fmt.Errorf("pin ID is empty")
	}
	return db.Get(pin)

}

//GetPin IDからピン留めを取得する
func GetPin(ID string) (*Pin, error) {
	if ID == "" {
		return nil, fmt.Errorf("ID is empty")
	}

	pin := &Pin{}
	if has, err := db.ID(ID).Get(pin); err != nil {
		return nil, err
	} else if !has {
		return nil, nil
	}

	return pin, nil
}

//GetPinsByChannelID あるチャンネルのピン留めを全部取得する
func GetPinsByChannelID(channelID string) (pins []*Pin, err error) {
	if channelID == "" {
		return nil, fmt.Errorf("ChannelID is empty")
	}

	err = db.Where("channel_id = ?", channelID).Find(&pins)
	return
}

//Delete ピン留めレコードを削除する
func (pin *Pin) Delete() (err error) {
	if pin.ID == "" {
		return fmt.Errorf("ID is empty")
	}

	_, err = db.Delete(pin)
	return
}
