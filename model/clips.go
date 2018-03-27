package model

import (
	"fmt"
	"github.com/traPtitech/traQ/utils/validator"
	"time"
)

// Clip clipの構造体
type Clip struct {
	UserID    string    `xorm:"char(36) pk"      validate:"uuid,required"`
	MessageID string    `xorm:"char(36) pk"      validate:"uuid,required"`
	CreatedAt time.Time `xorm:"created not null"`
}

// TableName dbの名前を指定する
func (clip *Clip) TableName() string {
	return "clips"
}

// Validate 構造体を検証します
func (clip *Clip) Validate() error {
	return validator.ValidateStruct(clip)
}

// Create clipをDBに入れる
func (clip *Clip) Create() (err error) {
	if err = clip.Validate(); err != nil {
		return
	}
	_, err = db.InsertOne(clip)
	return
}

// GetClippedMessages userIDがクリップしているメッセージの一覧を取得する
func GetClippedMessages(userID string) (messages []*Message, err error) {
	if userID == "" {
		return nil, fmt.Errorf("UserID is empty")
	}

	err = db.Table("clips").Join("INNER", "messages", "clips.message_id = messages.id").Where("clips.user_id = ? AND messages.is_deleted = false", userID).Find(&messages)
	return
}

// Delete clipを削除する
func (clip *Clip) Delete() (err error) {
	if err = clip.Validate(); err != nil {
		return
	}

	_, err = db.Delete(clip)
	return
}
