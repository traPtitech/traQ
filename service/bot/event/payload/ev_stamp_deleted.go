package payload

import (
	"time"

	"github.com/gofrs/uuid"
)

// StampDeleted STAMP_DELETEDイベントペイロード
type StampDeleted struct {
	Base
	ID uuid.UUID `json:"id"`
}

func MakeStampDeleted(et time.Time, stampID uuid.UUID) *StampDeleted {
	return &StampDeleted{
		Base: MakeBase(et),
		ID:   stampID,
	}
}
