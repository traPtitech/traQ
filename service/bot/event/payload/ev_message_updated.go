package payload

import (
	"time"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils/message"
)

// MessageUpdated MESSAGE_UPDATEDイベントペイロード
type MessageUpdated struct {
	Base
	Message Message `json:"message"`
}

func MakeMessageUpdated(et time.Time, m *model.Message, user model.UserInfo, parsed *message.ParseResult) *MessageUpdated {
	return &MessageUpdated{
		Base:    MakeBase(et),
		Message: MakeMessage(m, user, parsed.Embeddings, parsed.PlainText),
	}
}
