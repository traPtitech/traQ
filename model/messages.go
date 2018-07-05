package model

import (
	"errors"
	"github.com/jinzhu/gorm"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/utils/validator"
	"time"
)

//Message :データベースに格納するmessageの構造体
type Message struct {
	ID        string     `gorm:"type:char(36);primary_key" validate:"uuid,required"`
	UserID    string     `gorm:"type:char(36)"             validate:"uuid,required"`
	ChannelID string     `gorm:"type:char(36);index"       validate:"uuid,required"`
	Text      string     `gorm:"type:text"                 validate:"required"`
	CreatedAt time.Time  `gorm:"precision:6;index"`
	UpdatedAt time.Time  `gorm:"precision:6"`
	DeletedAt *time.Time `gorm:"precision:6;index"`
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

// BeforeCreate db.Create時に自動的に呼ばれます
func (m *Message) BeforeCreate(scope *gorm.Scope) error {
	m.ID = CreateUUID()
	return m.Validate()
}

// Validate 構造体を検証します
func (m *Message) Validate() error {
	return validator.ValidateStruct(m)
}

// CreateMessage メッセージを作成します
func CreateMessage(userID, channelID uuid.UUID, text string) (*Message, error) {
	m := &Message{
		UserID:    userID.String(),
		ChannelID: channelID.String(),
		Text:      text,
	}
	if err := db.Create(m).Error; err != nil {
		return nil, err
	}
	return m, nil
}

// UpdateMessage メッセージを更新します
func UpdateMessage(messageID uuid.UUID, text string) error {
	if len(text) == 0 {
		return errors.New("text is empty")
	}
	return db.Model(Message{ID: messageID.String()}).Update("text", text).Error
}

// DeleteMessage メッセージを削除します
func DeleteMessage(messageID uuid.UUID) error {
	return db.Delete(Message{ID: messageID.String()}).Error
}

// GetMessagesByChannelID 指定されたチャンネルのメッセージを取得します
func GetMessagesByChannelID(channelID uuid.UUID, limit, offset int) (list []*Message, err error) {
	if limit <= 0 {
		err = db.
			Where(Message{ChannelID: channelID.String()}).
			Order("created_at DESC").
			Offset(offset).
			Find(&list).
			Error
	} else {
		err = db.
			Where(Message{ChannelID: channelID.String()}).
			Order("created_at DESC").
			Offset(offset).
			Limit(limit).
			Find(&list).
			Error
	}
	return
}

// GetMessageByID messageIDで指定されたメッセージを取得します
func GetMessageByID(messageID uuid.UUID) (*Message, error) {
	message := &Message{}
	if err := db.Where(Message{ID: messageID.String()}).Take(message).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return message, nil
}
