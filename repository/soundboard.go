package repository

import (
	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/model"
)

type SoundboardRepository interface {
	CreateSoundboardItem(soundID uuid.UUID, soundName string, stampID *uuid.UUID, creatorID uuid.UUID) error
	GetAllSoundboardItems() ([]*model.SoundboardItem, error)
	GetSoundboardByCreatorID(creatorID uuid.UUID) ([]*model.SoundboardItem, error)
	UpdateSoundboardCreatorID(soundID uuid.UUID, creatorID uuid.UUID) error
	DeleteSoundboardItem(soundID uuid.UUID) error
}
