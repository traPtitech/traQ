package event

import (
	"github.com/gofrs/uuid"
)

// UserTagAddedPayload UserTagAdded のペイロード
type UserTagAddedPayload struct {
	UserID uuid.UUID `hub:"user_id"`
	TagID  uuid.UUID `hub:"tag_id"`
}

func (UserTagAddedPayload) TopicName() string { return UserTagAdded }

// UserTagUpdatedPayload UserTagUpdated のペイロード
type UserTagUpdatedPayload struct {
	UserID uuid.UUID `hub:"user_id"`
	TagID  uuid.UUID `hub:"tag_id"`
}

func (UserTagUpdatedPayload) TopicName() string { return UserTagUpdated }

// UserTagRemovedPayload UserTagRemoved のペイロード
type UserTagRemovedPayload struct {
	UserID uuid.UUID `hub:"user_id"`
	TagID  uuid.UUID `hub:"tag_id"`
}

func (UserTagRemovedPayload) TopicName() string { return UserTagRemoved }

var (
	_ Payload = UserTagAddedPayload{}
	_ Payload = UserTagUpdatedPayload{}
	_ Payload = UserTagRemovedPayload{}
)
