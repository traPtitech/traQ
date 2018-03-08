package model

import (
	"github.com/labstack/gommon/log"
	"time"
)

// MessageStamp : メッセージスタンプ構造体
type MessageStamp struct {
	MessageID string    `xorm:"char(36) unique(message_stamp_user) not null" json:"-"`
	StampID   string    `xorm:"char(36) unique(message_stamp_user) not null" json:"stampId"`
	UserID    string    `xorm:"char(36) unique(message_stamp_user) not null" json:"userId"`
	Count     int       `xorm:"int not null"                                 json:"count"`
	CreatedAt time.Time `xorm:"created"                                      json:"-"`
	UpdatedAt time.Time `xorm:"updated"                                      json:"-"`
}

// TableName : メッセージスタンプのテーブル
func (*MessageStamp) TableName() string {
	return "messages_stamps"
}

// AddStampToMessage : メッセージにスタンプを押します
func AddStampToMessage(messageID, stampID, userID string) (int, error) {
	_, err := db.Exec("INSERT INTO `messages_stamps` (`message_id`, `stamp_id`, `user_id`, `count`, `created_at`, `updated_at`) VALUES (?, ?, ?, 1, now(), now()) ON DUPLICATE KEY UPDATE `count` = `count` + 1, `updated_at` = now()", messageID, stampID, userID)
	if err != nil {
		return 0, err
	}
	count := 0
	if _, err := db.Table("messages_stamps").Where("message_id = ? AND stamp_id = ? AND user_id = ?", messageID, stampID, userID).Cols("count").Get(&count); err != nil {
		log.Error(err)
	}

	return count, nil
}

// RemoveStampFromMessage : メッセージからスタンプを消します
func RemoveStampFromMessage(messageID, stampID, userID string) error {
	_, err := db.Delete(&MessageStamp{MessageID: messageID, StampID: stampID, UserID: userID})
	if err != nil {
		return err
	}
	return nil
}

// GetMessageStamps : 指定したIDのメッセージのスタンプを取得します
func GetMessageStamps(messageID string) (stamps []*MessageStamp, err error) {
	err = db.Join("INNER", "stamps", "messages_stamps.stamp_id = stamps.id").Where("messages_stamps.message_id = ? AND stamps.is_deleted = false", messageID).Find(&stamps)
	return
}
