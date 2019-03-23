package repository

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
)

// MessageStampRepository メッセージスタンプリポジトリ
type MessageStampRepository interface {
	AddStampToMessage(messageID, stampID, userID uuid.UUID) (ms *model.MessageStamp, err error)
	RemoveStampFromMessage(messageID, stampID, userID uuid.UUID) (err error)
	GetMessageStamps(messageID uuid.UUID) (stamps []*model.MessageStamp, err error)
	GetUserStampHistory(userID uuid.UUID) (h []*model.UserStampHistory, err error)
}
