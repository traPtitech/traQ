package model

import (
	"github.com/jinzhu/gorm"
	"github.com/satori/go.uuid"
	"time"
)

// Pin ピン留めのレコード
type Pin struct {
	ID        string    `gorm:"type:char(36);primary_key"`
	MessageID string    `gorm:"type:char(36);unique"`
	Message   Message   `gorm:"association_autoupdate:false;association_autocreate:false"`
	UserID    string    `gorm:"type:char(36)"`
	CreatedAt time.Time `gorm:"precision:6"`
}

// TableName ピン留めテーブル名
func (pin *Pin) TableName() string {
	return "pins"
}

// CreatePin ピン留めレコードを追加する
func CreatePin(messageID, userID uuid.UUID) (uuid.UUID, error) {
	id := uuid.NewV4()
	p := &Pin{
		ID:        id.String(),
		MessageID: messageID.String(),
		UserID:    userID.String(),
	}
	if err := db.Create(p).Error; err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

// GetPin IDからピン留めを取得する
func GetPin(id uuid.UUID) (p *Pin, err error) {
	p = &Pin{}
	err = db.Preload("Message").Where(Pin{ID: id.String()}).Take(p).Error
	if err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, nil
		}
		return nil, err
	}
	return
}

// IsPinned 指定したメッセージがピン留めされているかを取得する
func IsPinned(messageID uuid.UUID) (bool, error) {
	p := &Pin{}
	err := db.Where(Pin{MessageID: messageID.String()}).Take(p).Error
	if err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// DeletePin ピン留めレコードを削除する
func DeletePin(id uuid.UUID) error {
	return db.Delete(Pin{ID: id.String()}).Error
}

// GetPinsByChannelID あるチャンネルのピン留めを全部取得する
func GetPinsByChannelID(channelID uuid.UUID) (pins []*Pin, err error) {
	err = db.
		Joins("INNER JOIN messages ON messages.id = pins.message_id").
		Where(Message{ChannelID: channelID.String()}).
		Preload("Message").
		Find(&pins).Error
	return
}
