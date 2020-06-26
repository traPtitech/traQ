package payload

import (
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils/message"
	"time"
)

// DirectMessageUpdated DIRECT_MESSAGE_UPDATEDイベントペイロード
type DirectMessageUpdated struct {
	Base
	Message Message `json:"message"`
}

func MakeDirectMessageUpdated(et time.Time, m *model.Message, user model.UserInfo, parsed *message.ParseResult) *DirectMessageUpdated {
	embedded, _ := message.ExtractEmbedding(m.Text)
	return &DirectMessageUpdated{
		Base:    MakeBase(et),
		Message: MakeMessage(m, user, embedded, parsed.PlainText),
	}
}
