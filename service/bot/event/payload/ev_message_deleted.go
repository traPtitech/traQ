package payload

import (
	"time"

	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/model"
)

// MessageDeleted MESSAGE_DELETEDイベントペイロード
type MessageDeleted struct {
	Base
	Message struct {
		ID        uuid.UUID `json:"id"`
		ChannelID uuid.UUID `json:"channelId"`
	} `json:"message"`
}

func MakeMessageDeleted(et time.Time, m *model.Message) *MessageDeleted {
	return &MessageDeleted{
		Base: MakeBase(et),
		Message: struct {
			ID        uuid.UUID `json:"id"`
			ChannelID uuid.UUID `json:"channelId"`
		}{
			ID:        m.ID,
			ChannelID: m.ChannelID,
		},
	}
}
