package webrtcv3

import (
	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/event"
)

// UserWebRTCv3StateChangedPayload UserWebRTCv3StateChanged のペイロード
type UserWebRTCv3StateChangedPayload struct {
	UserID    uuid.UUID         `hub:"user_id"`
	ChannelID uuid.UUID         `hub:"channel_id"`
	Sessions  map[string]string `hub:"sessions"`
}

func (UserWebRTCv3StateChangedPayload) TopicName() string { return event.UserWebRTCv3StateChanged }

var _ event.Payload = UserWebRTCv3StateChangedPayload{}
