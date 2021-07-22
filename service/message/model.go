package message

import (
	"encoding/json"
	"time"

	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/model"
)

type Message interface {
	GetID() uuid.UUID
	GetUserID() uuid.UUID
	GetChannelID() uuid.UUID
	GetText() string
	GetCreatedAt() time.Time
	GetUpdatedAt() time.Time
	GetStamps() []model.MessageStamp
	GetPin() *model.Pin

	json.Marshaler
}
