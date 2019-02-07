package model

import (
	"github.com/satori/go.uuid"
	"time"
)

// MessageStamp メッセージスタンプ構造体
type MessageStamp struct {
	MessageID uuid.UUID `gorm:"type:char(36);primary_key" json:"-"`
	StampID   uuid.UUID `gorm:"type:char(36);primary_key" json:"stampId"`
	UserID    uuid.UUID `gorm:"type:char(36);primary_key" json:"userId"`
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
	StampID  uuid.UUID `json:"stampId"`
	Datetime time.Time `json:"datetime"`
}

// AddStampToMessage メッセージにスタンプを押します
func AddStampToMessage(messageID, stampID, userID uuid.UUID) (*MessageStamp, error) {
	if messageID == uuid.Nil || stampID == uuid.Nil || userID == uuid.Nil {
		return nil, ErrNilID
	}

	err := db.
		Set("gorm:insert_option", "ON DUPLICATE KEY UPDATE count = count + 1, updated_at = now()").
		Create(&MessageStamp{MessageID: messageID, StampID: stampID, UserID: userID, Count: 1}).
		Error
	if err != nil {
		return nil, err
	}

	ms := &MessageStamp{}
	err = db.Where(&MessageStamp{MessageID: messageID, StampID: stampID, UserID: userID}).Take(ms).Error
	if err != nil {
		return nil, err
	}
	return ms, nil
}

// RemoveStampFromMessage メッセージからスタンプを消します
func RemoveStampFromMessage(messageID, stampID, userID uuid.UUID) error {
	if messageID == uuid.Nil || stampID == uuid.Nil || userID == uuid.Nil {
		return ErrNilID
	}
	return db.Where(&MessageStamp{MessageID: messageID, StampID: stampID, UserID: userID}).Delete(&MessageStamp{}).Error
}

// GetMessageStamps 指定したIDのメッセージのスタンプを取得します
func GetMessageStamps(messageID uuid.UUID) (stamps []*MessageStamp, err error) {
	if messageID == uuid.Nil {
		return nil, ErrNilID
	}
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
	if userID == uuid.Nil {
		return nil, ErrNilID
	}
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
