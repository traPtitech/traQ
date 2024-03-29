package payload

import (
	"time"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils/message"
)

// DirectMessageCreated DIRECT_MESSAGE_CREATEDイベントペイロード
type DirectMessageCreated struct {
	Base
	Message Message `json:"message"`
}

func MakeDirectMessageCreated(et time.Time, m *model.Message, user model.UserInfo, parsed *message.ParseResult) *DirectMessageCreated {
	embedded, _ := message.ExtractEmbedding(m.Text)
	return &DirectMessageCreated{
		Base:    MakeBase(et),
		Message: MakeMessage(m, user, embedded, parsed.PlainText),
	}
}
