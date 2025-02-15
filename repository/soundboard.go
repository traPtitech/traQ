package repository

import (
	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/model"
)

type SoundboardRepository interface {
	CreateSoundBoardItem(soundID uuid.UUID, soundName string, stampID *uuid.UUID, creatorID uuid.UUID) error
	GetAllSoundBoardItems() ([]*model.SoundboardItem, error)
	GetSoundboardByCreatorID(creatorID uuid.UUID) ([]*model.SoundboardItem, error)
	UpdateSoundboardCreatorID(soundID uuid.UUID, creatorID uuid.UUID) error
	DeleteSoundboardItem(soundID uuid.UUID) error
}
