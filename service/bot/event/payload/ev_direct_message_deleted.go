package payload

import (
	"time"

	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/model"
)

// DirectMessageDeleted DIRECT_MESSAGE_DELETEDイベントペイロード
type DirectMessageDeleted struct {
	Base
	Message struct {
		ID        uuid.UUID `json:"id"`
		UserID    uuid.UUID `json:"userId"`
		ChannelID uuid.UUID `json:"channelId"`
	} `json:"message"`
}

func MakeDirectMessageDeleted(et time.Time, m *model.Message) *DirectMessageDeleted {
	return &DirectMessageDeleted{
		Base: MakeBase(et),
		Message: struct {
			ID        uuid.UUID `json:"id"`
			UserID    uuid.UUID `json:"userId"`
			ChannelID uuid.UUID `json:"channelId"`
		}{
			ID:        m.ID,
			UserID:    m.UserID,
			ChannelID: m.ChannelID,
		},
	}
}
