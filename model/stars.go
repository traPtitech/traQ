package model

import "fmt"

// Star starの構造体
type Star struct {
	UserID    string `xorm:"char(36) pk"`
	ChannelID string `xorm:"char(36) pk"`
	CreatedAt string `xorm:"created not null"`
}

// TableName dbの名前を指定する
func (star *Star) TableName() string {
	return "stars"
}

// Create starをDBに入れる
func (star *Star) Create() error {
	if star.UserID == "" {
		return fmt.Errorf("UserID is empty")
	}

	if star.ChannelID == "" {
		return fmt.Errorf("ChannelID is empty")
	}

	if _, err := db.Insert(star); err != nil {
		return fmt.Errorf("Failed to create star object: %v", err)
	}
	return nil
}

// GetStaredChannels userIDがお気に入りしているチャンネルの一覧を取得する
func GetStaredChannels(userID string) ([]*Channel, error) {
	if userID == "" {
		return nil, fmt.Errorf("UserID is empty")
	}

	var channels []*Channel
	err := db.Table("stars").Join("LEFT", "channels", "stars.channel_id = channels.id").Where("stars.user_id = ? AND channels.is_deleted = false", userID).Find(&channels)
	if err != nil {
		return nil, fmt.Errorf("Failed to find stared channels :%v", err)
	}
	return channels, nil
}

// Delete starを削除する
func (star *Star) Delete() error {
	if star.UserID == "" {
		return fmt.Errorf("UserID is empty")
	}

	if star.ChannelID == "" {
		return fmt.Errorf("ChannelID is empty")
	}

	if _, err := db.Delete(star); err != nil {
		return fmt.Errorf("Failed to delete star: %v", err)
	}

	return nil
}
