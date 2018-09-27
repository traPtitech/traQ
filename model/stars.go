package model

import (
	"github.com/satori/go.uuid"
	"time"
)

// Star starの構造体
type Star struct {
	UserID    uuid.UUID `gorm:"type:char(36);primary_key"`
	ChannelID uuid.UUID `gorm:"type:char(36);primary_key"`
	CreatedAt time.Time `gorm:"precision:6"`
}

// TableName dbの名前を指定する
func (star *Star) TableName() string {
	return "stars"
}

// AddStar チャンネルをお気に入り登録します
func AddStar(userID, channelID uuid.UUID) error {
	// ユーザーからチャンネルが見えるかどうか
	ok, err := IsChannelAccessibleToUser(userID, channelID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFoundOrForbidden
	}

	if err := db.Create(&Star{UserID: userID, ChannelID: channelID}).Error; err != nil {
		if isMySQLDuplicatedRecordErr(err) {
			return nil
		}
		return err
	}
	return nil
}

// RemoveStar チャンネルのお気に入りを解除します
func RemoveStar(userID, channelID uuid.UUID) error {
	return db.Where(&Star{UserID: userID, ChannelID: channelID}).Delete(Star{}).Error
}

// GetStaredChannels userIDがお気に入りしているチャンネルIDを取得する
func GetStaredChannels(userID uuid.UUID) (channels []string, err error) {
	channels = make([]string, 0)
	err = db.Model(Star{}).Where(&Star{UserID: userID}).Pluck("channel_id", &channels).Error
	return
}
