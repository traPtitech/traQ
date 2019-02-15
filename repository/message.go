package repository

import (
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/model"
)

// MessageRepository メッセージリポジトリ
type MessageRepository interface {
	CreateMessage(userID, channelID uuid.UUID, text string) (*model.Message, error)
	UpdateMessage(messageID uuid.UUID, text string) error
	DeleteMessage(messageID uuid.UUID) error
	GetMessageByID(messageID uuid.UUID) (*model.Message, error)
	GetMessagesByChannelID(channelID uuid.UUID, limit, offset int) ([]*model.Message, error)
	SetMessageUnread(userID, messageID uuid.UUID) error
	GetUnreadMessagesByUserID(userID uuid.UUID) ([]*model.Message, error)
	DeleteUnreadsByMessageID(messageID uuid.UUID) error
	DeleteUnreadsByChannelID(channelID, userID uuid.UUID) error
	GetChannelLatestMessagesByUserID(userID uuid.UUID, limit int, subscribeOnly bool) ([]*model.Message, error)
}
