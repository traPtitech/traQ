package payload

import (
	"time"

	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/model"
)

// BotMessageStampsUpdated BOT_MESSAGE_STAMPS_UPDATEDイベントペイロード
type BotMessageStampsUpdated struct {
	Base
	MessageID uuid.UUID            `json:"messageId"`
	Stamps    []model.MessageStamp `json:"stamps"`
}

func MakeBotMessageStampsUpdated(eventTime time.Time, mid uuid.UUID, stamps []model.MessageStamp) *BotMessageStampsUpdated {
	return &BotMessageStampsUpdated{
		Base:      MakeBase(eventTime),
		MessageID: mid,
		Stamps:    stamps,
	}
}
