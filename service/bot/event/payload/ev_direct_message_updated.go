package payload

import (
	"time"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils/message"
)

// DirectMessageUpdated DIRECT_MESSAGE_UPDATEDイベントペイロード
type DirectMessageUpdated struct {
	Base
	Message Message `json:"message"`
}

func MakeDirectMessageUpdated(et time.Time, m *model.Message, user model.UserInfo, parsed *message.ParseResult) *DirectMessageUpdated {
	return &DirectMessageUpdated{
		Base:    MakeBase(et),
		Message: MakeMessage(m, user, parsed.Embeddings, parsed.PlainText),
	}
}
