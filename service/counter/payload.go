package counter

import (
	"time"

	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/event"
)

// UserOnlinePayload UserOnline のペイロード
type UserOnlinePayload struct {
	UserID   uuid.UUID `hub:"user_id"`
	Datetime time.Time `hub:"datetime"`
}

func (UserOnlinePayload) TopicName() string { return event.UserOnline }

// UserOfflinePayload UserOffline のペイロード
type UserOfflinePayload struct {
	UserID   uuid.UUID `hub:"user_id"`
	Datetime time.Time `hub:"datetime"`
}

func (UserOfflinePayload) TopicName() string { return event.UserOffline }

var (
	_ event.Payload = UserOnlinePayload{}
	_ event.Payload = UserOfflinePayload{}
)
