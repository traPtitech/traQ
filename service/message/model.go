package message

import (
	"encoding/json"
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
	"time"
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
