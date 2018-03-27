package model

import (
	"fmt"
	"github.com/traPtitech/traQ/utils/validator"
	"time"
)

// Star starの構造体
type Star struct {
	UserID    string    `xorm:"char(36) pk"      validate:"uuid,required"`
	ChannelID string    `xorm:"char(36) pk"      validate:"uuid,required"`
	CreatedAt time.Time `xorm:"created not null"`
}

// TableName dbの名前を指定する
func (star *Star) TableName() string {
	return "stars"
}

// Validate 構造体を検証します
func (star *Star) Validate() error {
	return validator.ValidateStruct(star)
}

// Create starをDBに入れる
func (star *Star) Create() (err error) {
	if err = star.Validate(); err != nil {
		return
	}
	_, err = db.InsertOne(star)
	return
}

// GetStaredChannels userIDがお気に入りしているチャンネルの一覧を取得する
func GetStaredChannels(userID string) (channels []*Channel, err error) {
	if userID == "" {
		return nil, fmt.Errorf("UserID is empty")
	}

	err = db.Table("stars").Join("INNER", "channels", "stars.channel_id = channels.id").Where("stars.user_id = ? AND channels.is_deleted = false", userID).Find(&channels)
	return
}

// Delete starを削除する
func (star *Star) Delete() (err error) {
	if err = star.Validate(); err != nil {
		return
	}

	_, err = db.Delete(star)
	return
}
