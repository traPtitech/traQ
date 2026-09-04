package gorm

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
)

func (repo *Repository) CreateUserFeatureFlag(userID uuid.UUID, featureFlagsJSON string) error {
	if userID == uuid.Nil {
		return repository.ErrNilID
	}
	featureFlag := &model.FeatureFlag{
		UserID:           userID,
		FeatureFlagsJSON: featureFlagsJSON,
	}
	if err := repo.db.Create(featureFlag).Error; err != nil {
		return convertError(err)
	}
	return nil
}

func (repo *Repository) UpdateUserFeatureFlag(userID uuid.UUID, featureFlagsJSON string) error {
	if userID == uuid.Nil {
		return repository.ErrNilID
	}

	if err := repo.db.Model(&model.FeatureFlag{UserID: userID}).Update("feature_flags_json", featureFlagsJSON).Error; err != nil {
		return convertError(err)
	}
	return nil
}

func (repo *Repository) GetFeatureFlagByUserID(userID uuid.UUID) (*model.FeatureFlag, error) {
	if userID == uuid.Nil {
		return nil, repository.ErrNilID
	}
	featureFlag := &model.FeatureFlag{}
	if err := repo.db.Take(featureFlag, &model.FeatureFlag{UserID: userID}).Error; err != nil {
		return nil, convertError(err)
	}
	return featureFlag, nil
}
