package event

import (
	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/model"
)

// StampPaletteCreatedPayload StampPaletteCreated のペイロード
type StampPaletteCreatedPayload struct {
	UserID         uuid.UUID           `hub:"user_id"`
	StampPaletteID uuid.UUID           `hub:"stamp_palette_id"`
	StampPalette   *model.StampPalette `hub:"stamp_palette"`
}

func (StampPaletteCreatedPayload) TopicName() string { return StampPaletteCreated }

// StampPaletteUpdatedPayload StampPaletteUpdated のペイロード
type StampPaletteUpdatedPayload struct {
	UserID         uuid.UUID `hub:"user_id"`
	StampPaletteID uuid.UUID `hub:"stamp_palette_id"`
}

func (StampPaletteUpdatedPayload) TopicName() string { return StampPaletteUpdated }

// StampPaletteDeletedPayload StampPaletteDeleted のペイロード
type StampPaletteDeletedPayload struct {
	UserID         uuid.UUID `hub:"user_id"`
	StampPaletteID uuid.UUID `hub:"stamp_palette_id"`
}

func (StampPaletteDeletedPayload) TopicName() string { return StampPaletteDeleted }

var (
	_ Payload = StampPaletteCreatedPayload{}
	_ Payload = StampPaletteUpdatedPayload{}
	_ Payload = StampPaletteDeletedPayload{}
)
