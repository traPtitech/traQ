package repository

import (
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/model"
)

// PinRepository ピンリポジトリ
type PinRepository interface {
	CreatePin(messageID, userID uuid.UUID) (uuid.UUID, error)
	GetPin(id uuid.UUID) (*model.Pin, error)
	IsPinned(messageID uuid.UUID) (bool, error)
	DeletePin(id uuid.UUID) error
	GetPinsByChannelID(channelID uuid.UUID) ([]*model.Pin, error)
}
