package event

import (
	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/model"
)

// UserCreatedPayload UserCreated のペイロード
type UserCreatedPayload struct {
	UserID uuid.UUID   `hub:"user_id"`
	User   *model.User `hub:"user"`
}

func (UserCreatedPayload) TopicName() string { return UserCreated }

// UserUpdatedPayload UserUpdated のペイロード
type UserUpdatedPayload struct {
	UserID uuid.UUID `hub:"user_id"`
}

func (UserUpdatedPayload) TopicName() string { return UserUpdated }

// UserIconUpdatedPayload UserIconUpdated のペイロード
type UserIconUpdatedPayload struct {
	UserID uuid.UUID `hub:"user_id"`
	FileID uuid.UUID `hub:"file_id"`
}

func (UserIconUpdatedPayload) TopicName() string { return UserIconUpdated }

// UserActivatedPayload UserActivated のペイロード
type UserActivatedPayload struct {
	User *model.User `hub:"user"`
}

func (UserActivatedPayload) TopicName() string { return UserActivated }

var (
	_ Payload = UserCreatedPayload{}
	_ Payload = UserUpdatedPayload{}
	_ Payload = UserIconUpdatedPayload{}
	_ Payload = UserActivatedPayload{}
)
