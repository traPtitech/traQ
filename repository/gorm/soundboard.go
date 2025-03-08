package gorm

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
)

func (repo *Repository) CreateSoundboardItem(soundID uuid.UUID, soundName string, stampID *uuid.UUID, creatorID uuid.UUID) error {
	return repo.db.Create(&model.SoundboardItem{
		ID:        soundID,
		Name:      soundName,
		StampID:   stampID,
		CreatorID: creatorID,
	}).Error
}

func (repo *Repository) GetAllSoundboardItems() ([]*model.SoundboardItem, error) {
	items := make([]*model.SoundboardItem, 0)
	if err := repo.db.Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (repo *Repository) GetSoundboardByCreatorID(creatorID uuid.UUID) ([]*model.SoundboardItem, error) {
	items := make([]*model.SoundboardItem, 0)
	if err := repo.db.Where(&model.SoundboardItem{CreatorID: creatorID}).Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (repo *Repository) UpdateSoundboardCreatorID(soundID uuid.UUID, creatorID uuid.UUID) error {
	return repo.db.Model(&model.SoundboardItem{}).Where("id = ?", soundID).Update("creator_id", creatorID).Error
}

func (repo *Repository) DeleteSoundboardItem(soundID uuid.UUID) error {
	return repo.db.Delete(&model.SoundboardItem{}, soundID).Error
}
