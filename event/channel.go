package event

import (
	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/model"
)

// ChannelCreatedPayload ChannelCreated のペイロード
type ChannelCreatedPayload struct {
	ChannelID uuid.UUID      `hub:"channel_id"`
	Channel   *model.Channel `hub:"channel"`
	Private   bool           `hub:"private"`
}

func (ChannelCreatedPayload) TopicName() string { return ChannelCreated }

// ChannelUpdatedPayload ChannelUpdated のペイロード
type ChannelUpdatedPayload struct {
	ChannelID uuid.UUID `hub:"channel_id"`
	Private   bool      `hub:"private"`
}

func (ChannelUpdatedPayload) TopicName() string { return ChannelUpdated }

// ChannelTopicUpdatedPayload ChannelTopicUpdated のペイロード
type ChannelTopicUpdatedPayload struct {
	ChannelID uuid.UUID `hub:"channel_id"`
	Topic     string    `hub:"topic"`
	UpdaterID uuid.UUID `hub:"updater_id"`
}

func (ChannelTopicUpdatedPayload) TopicName() string { return ChannelTopicUpdated }

// ChannelDeletedPayload ChannelDeleted のペイロード
type ChannelDeletedPayload struct {
	ChannelID uuid.UUID `hub:"channel_id"`
	Private   bool      `hub:"private"`
}

func (ChannelDeletedPayload) TopicName() string { return ChannelDeleted }

// ChannelReadPayload ChannelRead のペイロード
type ChannelReadPayload struct {
	UserID          uuid.UUID `hub:"user_id"`
	ChannelID       uuid.UUID `hub:"channel_id"`
	ReadMessagesNum int       `hub:"read_messages_num"`
}

func (ChannelReadPayload) TopicName() string { return ChannelRead }

// ChannelStaredPayload ChannelStared のペイロード
type ChannelStaredPayload struct {
	UserID    uuid.UUID `hub:"user_id"`
	ChannelID uuid.UUID `hub:"channel_id"`
}

func (ChannelStaredPayload) TopicName() string { return ChannelStared }

// ChannelUnstaredPayload ChannelUnstared のペイロード
type ChannelUnstaredPayload struct {
	UserID    uuid.UUID `hub:"user_id"`
	ChannelID uuid.UUID `hub:"channel_id"`
}

func (ChannelUnstaredPayload) TopicName() string { return ChannelUnstared }

// ChannelSubscribersChangedPayload ChannelSubscribersChanged のペイロード
type ChannelSubscribersChangedPayload struct {
	ChannelID     uuid.UUID   `hub:"channel_id"`
	SubscriberIDs []uuid.UUID `hub:"subscriber_ids"`
}

func (ChannelSubscribersChangedPayload) TopicName() string { return ChannelSubscribersChanged }

var (
	_ Payload = ChannelCreatedPayload{}
	_ Payload = ChannelUpdatedPayload{}
	_ Payload = ChannelTopicUpdatedPayload{}
	_ Payload = ChannelDeletedPayload{}
	_ Payload = ChannelReadPayload{}
	_ Payload = ChannelStaredPayload{}
	_ Payload = ChannelUnstaredPayload{}
	_ Payload = ChannelSubscribersChangedPayload{}
)
