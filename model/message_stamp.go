package model

import (
	"github.com/satori/go.uuid"
	"time"
)

// MessageStamp メッセージスタンプ構造体
type MessageStamp struct {
	MessageID string    `gorm:"type:char(36);primary_key" json:"-"`
	StampID   string    `gorm:"type:char(36);primary_key" json:"stampId"`
	UserID    string    `gorm:"type:char(36);primary_key" json:"userId"`
	Count     int       `                                 json:"count"`
	CreatedAt time.Time `gorm:"precision:6"               json:"createdAt"`
	UpdatedAt time.Time `gorm:"precision:6;index"         json:"updatedAt"`
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
func AddStampToMessage(messageID, stampID, userID uuid.UUID) (*MessageStamp, error) {
	m := messageID.String()
	s := stampID.String()
	u := userID.String()

	err := db.
		Set("gorm:insert_option", "ON DUPLICATE KEY UPDATE count = count + 1, updated_at = now()").
		Create(&MessageStamp{MessageID: m, StampID: s, UserID: u, Count: 1}).
		Error
	if err != nil {
		return nil, err
	}

	ms := &MessageStamp{}
	err = db.Where(&MessageStamp{MessageID: m, StampID: s, UserID: u}).Take(ms).Error
	if err != nil {
		return nil, err
	}
	return ms, nil
}

// RemoveStampFromMessage メッセージからスタンプを消します
func RemoveStampFromMessage(messageID, stampID, userID uuid.UUID) error {
	return db.Where(&MessageStamp{MessageID: messageID.String(), StampID: stampID.String(), UserID: userID.String()}).Delete(MessageStamp{}).Error
}

// GetMessageStamps 指定したIDのメッセージのスタンプを取得します
func GetMessageStamps(messageID uuid.UUID) (stamps []*MessageStamp, err error) {
	stamps = make([]*MessageStamp, 0)
	err = db.
		Joins("JOIN stamps ON messages_stamps.stamp_id = stamps.id AND messages_stamps.message_id = ?", messageID.String()).
		Order("messages_stamps.updated_at").
		Find(&stamps).
		Error
	return
}

// GetUserStampHistory 指定したユーザーのスタンプ履歴を最大50件取得します。
func GetUserStampHistory(userID uuid.UUID) (h []*UserStampHistory, err error) {
	h = make([]*UserStampHistory, 0)
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
