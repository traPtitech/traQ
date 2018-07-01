package model

import (
	"github.com/jinzhu/gorm"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/utils/validator"
	"time"
)

// Star starの構造体
type Star struct {
	ID        string    `gorm:"type:char(36);primary_key"`
	UserID    string    `gorm:"type:char(36);unique_index:user_channel" validate:"uuid,required"`
	ChannelID string    `gorm:"type:char(36);unique_index:user_channel" validate:"uuid,required"`
	CreatedAt time.Time `gorm:"precision:6"`
}

// TableName dbの名前を指定する
func (star *Star) TableName() string {
	return "stars"
}

// BeforeCreate db.Create時に自動的に呼ばれます
func (star *Star) BeforeCreate(scope *gorm.Scope) error {
	star.ID = CreateUUID()
	return star.Validate()
}

// Validate 構造体を検証します
func (star *Star) Validate() error {
	return validator.ValidateStruct(star)
}

// AddStar チャンネルをお気に入り登録します
func AddStar(userID, channelID uuid.UUID) (*Star, error) {
	s := &Star{
		UserID:    userID.String(),
		ChannelID: channelID.String(),
	}

	if err := db.Create(s).Error; err != nil {
		return nil, err
	}
	return s, nil
}

// RemoveStar チャンネルのお気に入りを解除します
func RemoveStar(userID, channelID uuid.UUID) error {
	return db.Where(Star{UserID: userID.String(), ChannelID: channelID.String()}).Delete(Star{}).Error
}

// GetStaredChannels userIDがお気に入りしているチャンネルの一覧を取得する
func GetStaredChannels(userID uuid.UUID) (channels []Channel, err error) {
	err = db.Joins("INNER JOIN stars ON stars.channel_id = channels.id").Where("stars.user_id = ?", userID.String()).Find(&channels).Error
	return
}
