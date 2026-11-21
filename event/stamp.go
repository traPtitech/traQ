package event

import (
	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/model"
)

// StampCreatedPayload StampCreated のペイロード
type StampCreatedPayload struct {
	StampID uuid.UUID    `hub:"stamp_id"`
	Stamp   *model.Stamp `hub:"stamp"`
}

func (StampCreatedPayload) TopicName() string { return StampCreated }

// StampUpdatedPayload StampUpdated のペイロード
type StampUpdatedPayload struct {
	StampID uuid.UUID    `hub:"stamp_id"`
	Stamp   *model.Stamp `hub:"stamp"`
}

func (StampUpdatedPayload) TopicName() string { return StampUpdated }

// StampDeletedPayload StampDeleted のペイロード
type StampDeletedPayload struct {
	StampID uuid.UUID `hub:"stamp_id"`
}

func (StampDeletedPayload) TopicName() string { return StampDeleted }

var (
	_ Payload = StampCreatedPayload{}
	_ Payload = StampUpdatedPayload{}
	_ Payload = StampDeletedPayload{}
)
