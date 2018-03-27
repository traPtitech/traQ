package model

import (
	"time"
)

// MessageStamp : メッセージスタンプ構造体
type MessageStamp struct {
	MessageID string    `xorm:"char(36) unique(message_stamp_user) not null" json:"-"`
	StampID   string    `xorm:"char(36) unique(message_stamp_user) not null" json:"stampId"`
	UserID    string    `xorm:"char(36) unique(message_stamp_user) not null" json:"userId"`
	Count     int       `xorm:"int not null"                                 json:"count"`
	CreatedAt time.Time `xorm:"created"                                      json:"createdAt"`
	UpdatedAt time.Time `xorm:"updated"                                      json:"updatedAt"`
}

// TableName : メッセージスタンプのテーブル
func (*MessageStamp) TableName() string {
	return "messages_stamps"
}

// AddStampToMessage : メッセージにスタンプを押します
func AddStampToMessage(messageID, stampID, userID string) (*MessageStamp, error) {
	_, err := db.Exec("INSERT INTO `messages_stamps` (`message_id`, `stamp_id`, `user_id`, `count`, `created_at`, `updated_at`) VALUES (?, ?, ?, 1, now(), now()) ON DUPLICATE KEY UPDATE `count` = `count` + 1, `updated_at` = now()", messageID, stampID, userID)
	if err != nil {
		return nil, err
	}
	ms := &MessageStamp{}
	if _, err := db.Table("messages_stamps").Where("message_id = ? AND stamp_id = ? AND user_id = ?", messageID, stampID, userID).Get(ms); err != nil {
		return nil, err
	}

	return ms, nil
}

// RemoveStampFromMessage : メッセージからスタンプを消します
func RemoveStampFromMessage(messageID, stampID, userID string) (err error) {
	_, err = db.Delete(&MessageStamp{MessageID: messageID, StampID: stampID, UserID: userID})
	return
}

// GetMessageStamps : 指定したIDのメッセージのスタンプを取得します
func GetMessageStamps(messageID string) (stamps []*MessageStamp, err error) {
	err = db.Join("INNER", "stamps", "messages_stamps.stamp_id = stamps.id").Where("messages_stamps.message_id = ? AND stamps.is_deleted = false", messageID).Find(&stamps)
	return
}
