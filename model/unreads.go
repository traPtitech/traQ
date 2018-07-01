package model

import (
	"github.com/satori/go.uuid"
	"time"
)

// Unread 未読レコード
type Unread struct {
	UserID    string    `gorm:"type:char(36);primary_key"`
	MessageID string    `gorm:"type:char(36);primary_key"`
	CreatedAt time.Time `gorm:"precision:6"`
}

// TableName テーブル名
func (unread *Unread) TableName() string {
	return "unreads"
}

// SetMessageUnread 指定したメッセージを未読にします
func SetMessageUnread(userID, messageID uuid.UUID) error {
	return db.Create(&Unread{UserID: userID.String(), MessageID: messageID.String()}).Error
}

// GetUnreadMessagesByUserID あるユーザーの未読メッセージをすべて取得
func GetUnreadMessagesByUserID(userID uuid.UUID) (unreads []*Message, err error) {
	err = db.Joins("INNER JOIN unreads ON unreads.message_id = messages.id AND unreads.user_id = ?", userID.String()).Find(&unreads).Error
	return
}

// DeleteUnreadsByMessageID 指定したメッセージIDの未読レコードを全て削除
func DeleteUnreadsByMessageID(messageID uuid.UUID) error {
	return db.Where(Unread{MessageID: messageID.String()}).Delete(Unread{}).Error
}

// DeleteUnreadsByChannelID 指定したチャンネルIDに存在する、指定したユーザーIDの未読レコードをすべて削除
func DeleteUnreadsByChannelID(channelID, userID uuid.UUID) error {
	return db.Exec("DELETE unreads FROM unreads INNER JOIN messages ON unreads.user_id = ? AND unreads.message_id = messages.id WHERE messages.channel_id = ?", userID.String(), channelID.String()).Error
}
