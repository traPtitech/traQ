package model

import (
	"github.com/satori/go.uuid"
	"time"
)

// Star starの構造体
type Star struct {
	UserID    string    `gorm:"type:char(36);primary_key"`
	ChannelID string    `gorm:"type:char(36);primary_key"`
	CreatedAt time.Time `gorm:"precision:6"`
}

// TableName dbの名前を指定する
func (star *Star) TableName() string {
	return "stars"
}

// AddStar チャンネルをお気に入り登録します
func AddStar(userID, channelID uuid.UUID) error {
	return db.Create(&Star{
		UserID:    userID.String(),
		ChannelID: channelID.String(),
	}).Error
}

// RemoveStar チャンネルのお気に入りを解除します
func RemoveStar(userID, channelID uuid.UUID) error {
	return db.Where(Star{UserID: userID.String(), ChannelID: channelID.String()}).Delete(Star{}).Error
}

// GetStaredChannels userIDがお気に入りしているチャンネルの一覧を取得する
func GetStaredChannels(userID uuid.UUID) (channels []*Channel, err error) {
	err = db.Joins("INNER JOIN stars ON stars.channel_id = channels.id").Where("stars.user_id = ?", userID.String()).Find(&channels).Error
	return
}
