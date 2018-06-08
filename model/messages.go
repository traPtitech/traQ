package model

import (
	"errors"
	"fmt"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/utils/validator"
	"time"
)

// ErrMessageAlreadyDeleted : メッセージエラー このメッセージは既に削除されている
var ErrMessageAlreadyDeleted = errors.New("this message has been deleted")

//Message :データベースに格納するmessageの構造体
type Message struct {
	ID        string    `xorm:"char(36) pk"       validate:"uuid,required"`
	UserID    string    `xorm:"char(36) not null" validate:"uuid,required"`
	ChannelID string    `xorm:"char(36)"          validate:"uuid,required"`
	Text      string    `xorm:"text not null"     validate:"required"`
	IsShared  bool      `xorm:"bool not null"`
	IsDeleted bool      `xorm:"bool not null"`
	CreatedAt time.Time `xorm:"created not null"`
	UpdaterID string    `xorm:"char(36) not null" validate:"uuid,required"`
	UpdatedAt time.Time `xorm:"updated not null"`
}

// GetID IDを返します
func (m *Message) GetID() uuid.UUID {
	return uuid.Must(uuid.FromString(m.ID))
}

// GetCID ChannelIDを返します
func (m *Message) GetCID() uuid.UUID {
	return uuid.Must(uuid.FromString(m.ChannelID))
}

// GetUID UserIDを返します
func (m *Message) GetUID() uuid.UUID {
	return uuid.Must(uuid.FromString(m.UserID))
}

// TableName DBの名前を指定するメソッド
func (m *Message) TableName() string {
	return "messages"
}

// Validate 構造体を検証します
func (m *Message) Validate() error {
	return validator.ValidateStruct(m)
}

// Create message構造体をDBに入れます
func (m *Message) Create() (err error) {
	m.ID = CreateUUID()
	m.IsDeleted = false
	m.UpdaterID = m.UserID

	if err = m.Validate(); err != nil {
		return
	}

	_, err = db.InsertOne(m)
	return
}

// Exists 指定されたメッセージが存在するかを判定します
func (m *Message) Exists() (bool, error) {
	if m.ID == "" {
		return false, fmt.Errorf("message ID is empty")
	}
	return db.Get(m)
}

// Update メッセージの内容を変更します
func (m *Message) Update() (err error) {
	if err = m.Validate(); err != nil {
		return
	}

	_, err = db.ID(m.ID).UseBool().Update(m)
	return
}

// IsPinned このメッセージがpin止めされているかどうかを調べる
func (m *Message) IsPinned() (bool, error) {
	if m.ID == "" {
		return false, ErrNotFound
	}

	return db.Get(&Pin{
		ChannelID: m.ChannelID,
		MessageID: m.ID,
	})
}

// GetMessagesByChannelID 指定されたチャンネルのメッセージを取得します
func GetMessagesByChannelID(channelID string, limit, offset int) (list []*Message, err error) {
	err = db.Where("channel_id = ? AND is_deleted = false", channelID).Desc("created_at").Limit(limit, offset).Find(&list)
	return
}

// GetMessageByID messageIDで指定されたメッセージを取得します
func GetMessageByID(messageID string) (*Message, error) {
	var message = &Message{}

	has, err := db.ID(messageID).Get(message)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, ErrNotFound
	} else if message.IsDeleted {
		return nil, ErrMessageAlreadyDeleted
	}

	return message, nil
}
