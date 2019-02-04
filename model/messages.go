package model

import (
	"errors"
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/utils/validator"
	"strings"
	"time"
)

// Message データベースに格納するmessageの構造体
type Message struct {
	ID        uuid.UUID  `gorm:"type:char(36);primary_key"`
	UserID    uuid.UUID  `gorm:"type:char(36)"`
	ChannelID uuid.UUID  `gorm:"type:char(36);index"`
	Text      string     `gorm:"type:text"                 validate:"required"`
	CreatedAt time.Time  `gorm:"precision:6;index"`
	UpdatedAt time.Time  `gorm:"precision:6"`
	DeletedAt *time.Time `gorm:"precision:6;index"`
}

// TableName DBの名前を指定するメソッド
func (m *Message) TableName() string {
	return "messages"
}

// BeforeCreate db.Create時に自動的に呼ばれます
func (m *Message) BeforeCreate(scope *gorm.Scope) error {
	m.ID = uuid.NewV4()
	return m.Validate()
}

// Validate 構造体を検証します
func (m *Message) Validate() error {
	return validator.ValidateStruct(m)
}

// Unread 未読レコード
type Unread struct {
	UserID    uuid.UUID `gorm:"type:char(36);primary_key"`
	MessageID uuid.UUID `gorm:"type:char(36);primary_key"`
	CreatedAt time.Time `gorm:"precision:6"`
}

// TableName テーブル名
func (unread *Unread) TableName() string {
	return "unreads"
}

// CreateMessage メッセージを作成します
func CreateMessage(userID, channelID uuid.UUID, text string) (*Message, error) {
	m := &Message{
		UserID:    userID,
		ChannelID: channelID,
		Text:      text,
	}
	if err := db.Create(m).Error; err != nil {
		return nil, err
	}
	return m, nil
}

// UpdateMessage メッセージを更新します
func UpdateMessage(messageID uuid.UUID, text string) error {
	if messageID == uuid.Nil {
		return ErrNilID
	}
	if len(text) == 0 {
		return errors.New("text is empty")
	}
	return db.Model(&Message{ID: messageID}).Update("text", text).Error
}

// DeleteMessage メッセージを削除します
func DeleteMessage(messageID uuid.UUID) error {
	if messageID == uuid.Nil {
		return ErrNilID
	}
	return db.Delete(&Message{ID: messageID}).Error
}

// GetMessagesByChannelID 指定されたチャンネルのメッセージを取得します
func GetMessagesByChannelID(channelID uuid.UUID, limit, offset int) (arr []*Message, err error) {
	if channelID == uuid.Nil {
		return nil, ErrNilID
	}
	arr = make([]*Message, 0)
	err = db.
		Where(&Message{ChannelID: channelID}).
		Order("created_at DESC").
		Scopes(limitAndOffset(limit, offset)).
		Find(&arr).
		Error
	return arr, err
}

// GetMessageByID messageIDで指定されたメッセージを取得します
func GetMessageByID(messageID uuid.UUID) (*Message, error) {
	if messageID == uuid.Nil {
		return nil, ErrNilID
	}
	message := &Message{}
	if err := db.Where(&Message{ID: messageID}).Take(message).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return message, nil
}

// SetMessageUnread 指定したメッセージを未読にします
func SetMessageUnread(userID, messageID uuid.UUID) error {
	if userID == uuid.Nil || messageID == uuid.Nil {
		return ErrNilID
	}
	return db.Create(&Unread{UserID: userID, MessageID: messageID}).Error
}

// GetUnreadMessagesByUserID あるユーザーの未読メッセージをすべて取得
func GetUnreadMessagesByUserID(userID uuid.UUID) (unreads []*Message, err error) {
	err = db.
		Joins("INNER JOIN unreads ON unreads.message_id = messages.id AND unreads.user_id = ?", userID.String()).
		Order("messages.created_at").
		Find(&unreads).
		Error
	return
}

// DeleteUnreadsByMessageID 指定したメッセージIDの未読レコードを全て削除
func DeleteUnreadsByMessageID(messageID uuid.UUID) error {
	if messageID == uuid.Nil {
		return ErrNilID
	}
	return db.Where(&Unread{MessageID: messageID}).Delete(Unread{}).Error
}

// DeleteUnreadsByChannelID 指定したチャンネルIDに存在する、指定したユーザーIDの未読レコードをすべて削除
func DeleteUnreadsByChannelID(channelID, userID uuid.UUID) error {
	return db.Exec("DELETE unreads FROM unreads INNER JOIN messages ON unreads.user_id = ? AND unreads.message_id = messages.id WHERE messages.channel_id = ?", userID.String(), channelID.String()).Error
}

// GetChannelLatestMessagesByUserID 指定したユーザーが閲覧可能な全てのチャンネルの最新のメッセージの一覧を取得します
func GetChannelLatestMessagesByUserID(userID uuid.UUID, limit int, subscribeOnly bool) ([]*Message, error) {
	var query string
	switch {
	case subscribeOnly:
		query = `
SELECT m.id, m.user_id, m.channel_id, m.text, m.created_at, m.updated_at, m.deleted_at
FROM (
       SELECT ROW_NUMBER() OVER(PARTITION BY m.channel_id ORDER BY m.created_at DESC) AS r,
              m.*
       FROM messages m
       WHERE m.deleted_at IS NULL
     ) m
       INNER JOIN channels c ON m.channel_id = c.id
       INNER JOIN (SELECT channel_id
                   FROM users_subscribe_channels
                   WHERE user_id = 'USER_ID'
                   UNION
                   SELECT channel_id
                   FROM users_private_channels
                   WHERE user_id = 'USER_ID') s ON s.channel_id = m.channel_id
WHERE m.r = 1 AND c.deleted_at IS NULL
ORDER BY m.created_at DESC
`
	default:
		query = `
SELECT m.id, m.user_id, m.channel_id, m.text, m.created_at, m.updated_at, m.deleted_at
FROM (
       SELECT ROW_NUMBER() OVER(PARTITION BY m.channel_id ORDER BY m.created_at DESC) AS r,
              m.*
       FROM messages m
       WHERE m.deleted_at IS NULL
     ) m
       INNER JOIN channels c ON m.channel_id = c.id
       LEFT JOIN users_private_channels upc ON upc.channel_id = m.channel_id
WHERE m.r = 1 AND c.deleted_at IS NULL AND (c.is_public = true OR upc.user_id = 'USER_ID')
ORDER BY m.created_at DESC
`
	}

	query = strings.Replace(query, "USER_ID", userID.String(), -1)
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	result := make([]*Message, 0)
	err := db.Raw(query).Scan(&result).Error
	return result, err
}
