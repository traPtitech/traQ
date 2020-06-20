package payload

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
	"time"
)

// MessageDeleted MESSAGE_DELETEDイベントペイロード
type MessageDeleted struct {
	// 用意されている Base を使いたい気分になる
	EventTime time.Time `json: "eventTime"`
	/* Base */
	Message MessageIDs `json:"message"`
	/* Message Message `json:"message"` */
}

// 用意されている Message を使いたい気分になる
// が、User などを指定しなくてもいいのかわからず
// こういう風に構造体を定義する方法で書くべきですか?
type MessageIDs struct {
	ID        uuid.UUID `json:"id"`
	ChannelID uuid.UUID `json:"channelId"`
}

func MakeMessageDeleted(et time.Time, m *model.Message) *MessageDeleted {
	return &MessageDeleted{
		EventTime: et,
		/* Base:    MakeBase(et); */
		Message:   MessageIDs{
		/* Message: Message{ */
			ID:        m.ID,
			ChannelID: m.ChannelID,
		},
	}
}