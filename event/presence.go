package event

import (
	"time"

	"github.com/gofrs/uuid"
)

// UserOnlinePayload UserOnline のペイロード
type UserOnlinePayload struct {
	UserID   uuid.UUID `hub:"user_id"`
	Datetime time.Time `hub:"datetime"`
}

func (UserOnlinePayload) TopicName() string { return UserOnline }

// UserOfflinePayload UserOffline のペイロード
type UserOfflinePayload struct {
	UserID   uuid.UUID `hub:"user_id"`
	Datetime time.Time `hub:"datetime"`
}

func (UserOfflinePayload) TopicName() string { return UserOffline }

var (
	_ Payload = UserOnlinePayload{}
	_ Payload = UserOfflinePayload{}
)
