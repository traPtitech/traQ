package payload

import (
	"time"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils/message"
)

// MessageCreated MESSAGE_CREATEDイベントペイロード
type MessageCreated struct {
	Base
	Message Message `json:"message"`
}

func MakeMessageCreated(et time.Time, m *model.Message, user model.UserInfo, parsed *message.ParseResult) *MessageCreated {
	embedded, _ := message.ExtractEmbedding(m.Text)
	return &MessageCreated{
		Base:    MakeBase(et),
		Message: MakeMessage(m, user, embedded, parsed.PlainText),
	}
}
