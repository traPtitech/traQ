package event

import (
	"github.com/gofrs/uuid"
)

// QallSoundboardItemCreatedPayload QallSoundboardItemCreated のペイロード
type QallSoundboardItemCreatedPayload struct {
	SoundID   uuid.UUID `hub:"sound_id"`
	Name      string    `hub:"name"`
	CreatorID uuid.UUID `hub:"creator_id"`
}

func (QallSoundboardItemCreatedPayload) TopicName() string { return QallSoundboardItemCreated }

// QallSoundboardItemDeletedPayload QallSoundboardItemDeleted のペイロード
type QallSoundboardItemDeletedPayload struct {
	SoundID uuid.UUID `hub:"sound_id"`
}

func (QallSoundboardItemDeletedPayload) TopicName() string { return QallSoundboardItemDeleted }

var (
	_ Payload = QallSoundboardItemCreatedPayload{}
	_ Payload = QallSoundboardItemDeletedPayload{}
)
