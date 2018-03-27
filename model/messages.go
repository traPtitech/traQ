package model

import (
	"errors"
	"fmt"
	"time"
)

// ErrMessageAlreadyDeleted : メッセージエラー このメッセージは既に削除されている
var ErrMessageAlreadyDeleted = errors.New("this message has been deleted")

//Message :データベースに格納するmessageの構造体
type Message struct {
	ID        string    `xorm:"char(36) pk"`
	UserID    string    `xorm:"char(36) not null"`
	ChannelID string    `xorm:"char(36)"`
	Text      string    `xorm:"text not null"`
	IsShared  bool      `xorm:"bool not null"`
	IsDeleted bool      `xorm:"bool not null"`
	CreatedAt time.Time `xorm:"created not null"`
	UpdaterID string    `xorm:"char(36) not null"`
	UpdatedAt time.Time `xorm:"updated not null"`
}

// TableName DBの名前を指定するメソッド
func (m *Message) TableName() string {
	return "messages"
}

// Create message構造体をDBに入れます
func (m *Message) Create() error {
	if m.UserID == "" {
		return fmt.Errorf("userID is empty")
	}

	if m.Text == "" {
		return fmt.Errorf("text is empty")
	}

	m.ID = CreateUUID()
	m.IsDeleted = false
	m.UpdaterID = m.UserID

	if _, err := db.Insert(m); err != nil {
		return err
	}
	return nil
}

// Exists 指定されたメッセージが存在するかを判定します
func (m *Message) Exists() (bool, error) {
	if m.ID == "" {
		return false, fmt.Errorf("message ID is empty")
	}
	return db.Get(m)
}

// Update メッセージの内容を変更します
func (m *Message) Update() error {
	_, err := db.ID(m.ID).UseBool().Update(m)
	if err != nil {
		return fmt.Errorf("failed to update this message: %v", err)
	}
	return nil
}

// IsPinned このメッセージがpin止めされているかどうかを調べる
func (m *Message) IsPinned() (bool, error) {
	if m.ID == "" {
		return false, ErrNotFound
	}

	p := &Pin{
		ChannelID: m.ChannelID,
		MessageID: m.ID,
	}

	return db.Get(p)
}

// GetMessagesByChannelID 指定されたチャンネルのメッセージを取得します
func GetMessagesByChannelID(channelID string, limit, offset int) ([]*Message, error) {
	var messageList []*Message
	err := db.Where("channel_id = ? AND is_deleted = false", channelID).Desc("created_at").Limit(limit, offset).Find(&messageList)
	if err != nil {
		return nil, fmt.Errorf("failed to find messages: %v", err)
	}

	return messageList, nil
}

// GetMessageByID messageIDで指定されたメッセージを取得します
func GetMessageByID(messageID string) (*Message, error) {
	var message = &Message{}
	has, err := db.ID(messageID).Get(message)

	if err != nil {
		return nil, fmt.Errorf("failed to find message: %v", err)
	}
	if has == false {
		return nil, ErrNotFound
	}
	if message.IsDeleted {
		return nil, ErrMessageAlreadyDeleted
	}

	return message, nil
}
