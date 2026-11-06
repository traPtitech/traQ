package qall

import (
	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/event"
)

// QallRoomStateChangedPayload QallRoomStateChanged のペイロード
type QallRoomStateChangedPayload struct { //nolint:revive // あとで fix したいから #<PR番号> で暫定的に入れている
	RoomStates []RoomWithParticipants `hub:"roomStates"`
}

func (QallRoomStateChangedPayload) TopicName() string { return event.QallRoomStateChanged }

// QallSoundboardItemCreatedPayload QallSoundboardItemCreated のペイロード
type QallSoundboardItemCreatedPayload struct { //nolint:revive // あとで fix したいから #<PR番号> で暫定的に入れている
	SoundID   uuid.UUID `hub:"sound_id"`
	Name      string    `hub:"name"`
	CreatorID uuid.UUID `hub:"creator_id"`
}

func (QallSoundboardItemCreatedPayload) TopicName() string { return event.QallSoundboardItemCreated }

// QallSoundboardItemDeletedPayload QallSoundboardItemDeleted のペイロード
type QallSoundboardItemDeletedPayload struct { //nolint:revive // あとで fix したいから #<PR番号> で暫定的に入れている
	SoundID uuid.UUID `hub:"sound_id"`
}

func (QallSoundboardItemDeletedPayload) TopicName() string { return event.QallSoundboardItemDeleted }

var (
	_ event.Payload = QallRoomStateChangedPayload{}
	_ event.Payload = QallSoundboardItemCreatedPayload{}
	_ event.Payload = QallSoundboardItemDeletedPayload{}
)
