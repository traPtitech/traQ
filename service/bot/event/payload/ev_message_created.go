package payload

import (
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils/message"
)

// MessageCreated MESSAGE_CREATEDイベントペイロード
type MessageCreated struct {
	Base
	Message Message `json:"message"`
}

func MakeMessageCreated(m *model.Message, user model.UserInfo, parsed *message.ParseResult) *MessageCreated {
	embedded, _ := message.ExtractEmbedding(m.Text)
	return &MessageCreated{
		Base:    MakeBase(),
		Message: MakeMessage(m, user, embedded, parsed.PlainText),
	}
}
