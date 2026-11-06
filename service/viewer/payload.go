package viewer

import (
	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/event"
)

// UserViewStateChangedPayload UserViewStateChanged のペイロード
type UserViewStateChangedPayload struct {
	UserID     uuid.UUID                   `hub:"user_id"`
	ViewStates map[string]StateWithChannel `hub:"view_states"`
}

func (UserViewStateChangedPayload) TopicName() string { return event.UserViewStateChanged }

// ChannelViewersChangedPayload ChannelViewersChanged のペイロード
type ChannelViewersChangedPayload struct {
	ChannelID uuid.UUID                   `hub:"channel_id"`
	Viewers   map[uuid.UUID]StateWithTime `hub:"viewers"`
}

func (ChannelViewersChangedPayload) TopicName() string { return event.ChannelViewersChanged }

var (
	_ event.Payload = UserViewStateChangedPayload{}
	_ event.Payload = ChannelViewersChangedPayload{}
)
