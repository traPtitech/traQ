package event

import (
	"time"

	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils/message"
)

// MessageCreatedPayload MessageCreated のペイロード
type MessageCreatedPayload struct {
	MessageID   uuid.UUID            `hub:"message_id"`
	Message     *model.Message       `hub:"message"`
	ParseResult *message.ParseResult `hub:"parse_result"`
}

func (MessageCreatedPayload) TopicName() string { return MessageCreated }

// MessageUpdatedPayload MessageUpdated のペイロード
type MessageUpdatedPayload struct {
	MessageID  uuid.UUID      `hub:"message_id"`
	OldMessage *model.Message `hub:"old_message"`
	Message    *model.Message `hub:"message"`
}

func (MessageUpdatedPayload) TopicName() string { return MessageUpdated }

// MessageDeletedPayload MessageDeleted のペイロード
type MessageDeletedPayload struct {
	MessageID      uuid.UUID       `hub:"message_id"`
	Message        *model.Message  `hub:"message"`
	DeletedUnreads []*model.Unread `hub:"deleted_unreads"`
}

func (MessageDeletedPayload) TopicName() string { return MessageDeleted }

// MessageUnreadPayload MessageUnread のペイロード
type MessageUnreadPayload struct {
	MessageID  uuid.UUID `hub:"message_id"`
	UserID     uuid.UUID `hub:"user_id"`
	Noticeable bool      `hub:"noticeable"`
}

func (MessageUnreadPayload) TopicName() string { return MessageUnread }

// MessageStampedPayload MessageStamped のペイロード
type MessageStampedPayload struct {
	MessageID uuid.UUID `hub:"message_id"`
	UserID    uuid.UUID `hub:"user_id"`
	StampID   uuid.UUID `hub:"stamp_id"`
	Count     int       `hub:"count"`
	CreatedAt time.Time `hub:"created_at"`
}

func (MessageStampedPayload) TopicName() string { return MessageStamped }

// MessageUnstampedPayload MessageUnstamped のペイロード
type MessageUnstampedPayload struct {
	MessageID uuid.UUID `hub:"message_id"`
	UserID    uuid.UUID `hub:"user_id"`
	StampID   uuid.UUID `hub:"stamp_id"`
}

func (MessageUnstampedPayload) TopicName() string { return MessageUnstamped }

// MessagePinnedPayload MessagePinned のペイロード
type MessagePinnedPayload struct {
	MessageID uuid.UUID `hub:"message_id"`
	ChannelID uuid.UUID `hub:"channel_id"`
}

func (MessagePinnedPayload) TopicName() string { return MessagePinned }

// MessageUnpinnedPayload MessageUnpinned のペイロード
type MessageUnpinnedPayload struct {
	MessageID uuid.UUID `hub:"message_id"`
	ChannelID uuid.UUID `hub:"channel_id"`
}

func (MessageUnpinnedPayload) TopicName() string { return MessageUnpinned }

// MessageCitedPayload MessageCited のペイロード
type MessageCitedPayload struct {
	MessageID uuid.UUID      `hub:"message_id"`
	Message   *model.Message `hub:"message"`
	CitedIDs  []uuid.UUID    `hub:"cited_ids"`
}

func (MessageCitedPayload) TopicName() string { return MessageCited }

var (
	_ Payload = MessageCreatedPayload{}
	_ Payload = MessageUpdatedPayload{}
	_ Payload = MessageDeletedPayload{}
	_ Payload = MessageUnreadPayload{}
	_ Payload = MessageStampedPayload{}
	_ Payload = MessageUnstampedPayload{}
	_ Payload = MessagePinnedPayload{}
	_ Payload = MessageUnpinnedPayload{}
	_ Payload = MessageCitedPayload{}
)
