package model

import (
	"fmt"
	"github.com/traPtitech/traQ/utils/validator"
)

//Unread 未読レコード
type Unread struct {
	UserID    string `xorm:"char(36) not null pk" validate:"uuid,required"`
	MessageID string `xorm:"char(36) not null pk" validate:"uuid,required"`
}

//TableName テーブル名
func (unread *Unread) TableName() string {
	return "unreads"
}

// Validate 構造体を検証します
func (unread *Unread) Validate() error {
	return validator.ValidateStruct(unread)
}

//Create レコード作成
func (unread *Unread) Create() (err error) {
	if err = unread.Validate(); err != nil {
		return err
	}
	_, err = db.InsertOne(unread)
	return
}

//Delete レコード削除
func (unread *Unread) Delete() (err error) {
	if err = unread.Validate(); err != nil {
		return err
	}
	_, err = db.Delete(unread)
	return
}

//GetUnreadsByUserID あるユーザーの未読レコードをすべて取得
func GetUnreadsByUserID(userID string) (unreads []*Unread, err error) {
	if userID == "" {
		return nil, fmt.Errorf("userID is empty")
	}

	err = db.Where("user_id = ?", userID).Find(&unreads)
	return
}

//DeleteUnreadsByMessageID 指定したメッセージIDの未読レコードを全て削除
func DeleteUnreadsByMessageID(messageID string) (err error) {
	if messageID == "" {
		return fmt.Errorf("messageID is empty")
	}

	_, err = db.Delete(&Unread{MessageID: messageID})
	return
}
