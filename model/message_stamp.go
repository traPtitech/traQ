package model

import (
	"github.com/satori/go.uuid"
	"time"
)

// MessageStamp メッセージスタンプ構造体
type MessageStamp struct {
	MessageID string    `gorm:"type:char(36);unique_index:message_stamp_user" json:"-"`
	StampID   string    `gorm:"type:char(36);unique_index:message_stamp_user" json:"stampId"`
	UserID    string    `gorm:"type:char(36);unique_index:message_stamp_user" json:"userId"`
	Count     int       `                                                     json:"count"`
	CreatedAt time.Time `gorm:"precision:6"                                   json:"createdAt"`
	UpdatedAt time.Time `gorm:"precision:6"                                   json:"updatedAt"`
}

// TableName メッセージスタンプのテーブル
func (*MessageStamp) TableName() string {
	return "messages_stamps"
}

// UserStampHistory スタンプ履歴構造体
type UserStampHistory struct {
	StampID  string    `json:"stampId"`
	Datetime time.Time `json:"datetime"`
}

// AddStampToMessage メッセージにスタンプを押します
func AddStampToMessage(messageID, stampID, userID string) (*MessageStamp, error) {
	err := db.
		Set("gorm:insert_option", "ON DUPLICATE KEY UPDATE count = count + 1, updated_at = now()").
		Create(&MessageStamp{MessageID: messageID, StampID: stampID, UserID: userID, Count: 1}).
		Error
	if err != nil {
		return nil, err
	}

	ms := &MessageStamp{}
	err = db.Where(MessageStamp{MessageID: messageID, StampID: stampID, UserID: userID}).Take(ms).Error
	if err != nil {
		return nil, err
	}
	return ms, nil
}

// RemoveStampFromMessage メッセージからスタンプを消します
func RemoveStampFromMessage(messageID, stampID, userID string) (err error) {
	err = db.Where(MessageStamp{MessageID: messageID, StampID: stampID, UserID: userID}).Delete(MessageStamp{}).Error
	return
}

// GetMessageStamps 指定したIDのメッセージのスタンプを取得します
func GetMessageStamps(messageID string) (stamps []MessageStamp, err error) {
	err = db.
		Joins("JOIN stamps ON messages_stamps.stamp_id = stamps.id AND messages_stamps.message_id = ? AND stamps.deleted_at IS NULL", messageID).
		Find(&stamps).
		Error
	return
}

// GetUserStampHistory 指定したユーザーのスタンプ履歴を最大50件取得します。
func GetUserStampHistory(userID uuid.UUID) (h []UserStampHistory, err error) {
	err = db.
		Table("messages_stamps").
		Where("user_id = ?", userID.String()).
		Group("stamp_id").
		Select("stamp_id, max(updated_at) AS datetime").
		Order("datetime DESC").
		Limit(50).
		Scan(&h).
		Error
	return
}
