package event

import (
	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/service/message"
)

// MessageStampsUpdatedPayload MessageStampsUpdated のペイロード
type MessageStampsUpdatedPayload struct {
	MessageID uuid.UUID       `hub:"message_id"`
	Message   message.Message `hub:"message"`
}

func (MessageStampsUpdatedPayload) TopicName() string { return MessageStampsUpdated }

var _ Payload = MessageStampsUpdatedPayload{}
