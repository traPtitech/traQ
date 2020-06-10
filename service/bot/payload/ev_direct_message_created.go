package payload

import (
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils/message"
)

// DirectMessageCreated DIRECT_MESSAGE_CREATEDイベントペイロード
type DirectMessageCreated struct {
	Base
	Message Message `json:"message"`
}

func MakeDirectMessageCreated(m *model.Message, user model.UserInfo, embedded []*message.EmbeddedInfo, parsed *message.ParseResult) *DirectMessageCreated {
	return &DirectMessageCreated{
		Base:    MakeBase(),
		Message: MakeMessage(m, user, embedded, parsed.PlainText),
	}
}
