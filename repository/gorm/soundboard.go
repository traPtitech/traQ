package gorm

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
)

func (repo *Repository) CreateSoundboardItem(ctx context.Context, args repository.CreateSoundboardItemArgs) error {
	return repo.db.WithContext(ctx).Create(&model.SoundboardItem{
		ID:        args.SoundID,
		Name:      args.SoundName,
		StampID:   args.StampID,
		CreatorID: args.CreatorID,
	}).Error
}

func (repo *Repository) GetAllSoundboardItems(ctx context.Context) ([]*model.SoundboardItem, error) {
	items := make([]*model.SoundboardItem, 0)
	if err := repo.db.WithContext(ctx).Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (repo *Repository) GetSoundboardByCreatorID(ctx context.Context, creatorID uuid.UUID) ([]*model.SoundboardItem, error) {
	items := make([]*model.SoundboardItem, 0)
	if err := repo.db.WithContext(ctx).Where(&model.SoundboardItem{CreatorID: creatorID}).Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (repo *Repository) UpdateSoundboardCreatorID(ctx context.Context, soundID uuid.UUID, creatorID uuid.UUID) error {
	return repo.db.WithContext(ctx).Model(&model.SoundboardItem{}).Where("id = ?", soundID).Update("creator_id", creatorID).Error
}

func (repo *Repository) DeleteSoundboardItem(ctx context.Context, soundID uuid.UUID) error {
	return repo.db.WithContext(ctx).Delete(&model.SoundboardItem{}, soundID).Error
}
