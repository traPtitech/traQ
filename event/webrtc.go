package event

import (
	"github.com/gofrs/uuid"
)

// UserWebRTCv3StateChangedPayload UserWebRTCv3StateChanged のペイロード
type UserWebRTCv3StateChangedPayload struct {
	UserID    uuid.UUID         `hub:"user_id"`
	ChannelID uuid.UUID         `hub:"channel_id"`
	Sessions  map[string]string `hub:"sessions"`
}

func (UserWebRTCv3StateChangedPayload) TopicName() string { return UserWebRTCv3StateChanged }

var _ Payload = UserWebRTCv3StateChangedPayload{}
