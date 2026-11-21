package event

import (
	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/model"
)

// BotCreatedPayload BotCreated のペイロード
type BotCreatedPayload struct {
	BotID uuid.UUID  `hub:"bot_id"`
	Bot   *model.Bot `hub:"bot"`
}

func (BotCreatedPayload) TopicName() string { return BotCreated }

// BotUpdatedPayload BotUpdated のペイロード
type BotUpdatedPayload struct {
	BotID uuid.UUID `hub:"bot_id"`
}

func (BotUpdatedPayload) TopicName() string { return BotUpdated }

// BotDeletedPayload BotDeleted のペイロード
type BotDeletedPayload struct {
	BotID uuid.UUID `hub:"bot_id"`
}

func (BotDeletedPayload) TopicName() string { return BotDeleted }

// BotStateChangedPayload BotStateChanged のペイロード
type BotStateChangedPayload struct {
	BotID uuid.UUID      `hub:"bot_id"`
	State model.BotState `hub:"state"`
}

func (BotStateChangedPayload) TopicName() string { return BotStateChanged }

// BotPingRequestPayload BotPingRequest のペイロード
type BotPingRequestPayload struct {
	BotID uuid.UUID  `hub:"bot_id"`
	Bot   *model.Bot `hub:"bot"`
}

func (BotPingRequestPayload) TopicName() string { return BotPingRequest }

// BotJoinedPayload BotJoined のペイロード
type BotJoinedPayload struct {
	BotID     uuid.UUID `hub:"bot_id"`
	ChannelID uuid.UUID `hub:"channel_id"`
}

func (BotJoinedPayload) TopicName() string { return BotJoined }

// BotLeftPayload BotLeft のペイロード
type BotLeftPayload struct {
	BotID     uuid.UUID `hub:"bot_id"`
	ChannelID uuid.UUID `hub:"channel_id"`
}

func (BotLeftPayload) TopicName() string { return BotLeft }

var (
	_ Payload = BotCreatedPayload{}
	_ Payload = BotUpdatedPayload{}
	_ Payload = BotDeletedPayload{}
	_ Payload = BotStateChangedPayload{}
	_ Payload = BotPingRequestPayload{}
	_ Payload = BotJoinedPayload{}
	_ Payload = BotLeftPayload{}
)
