package gorm

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
)

func (repo *Repository) GetFeatureFlagByUserID(userID uuid.UUID) (*model.FeatureFlag, error) {
	if userID == uuid.Nil {
		return nil, repository.ErrNotFound
	}
	featureFlag := &model.FeatureFlag{}
	if err := repo.db.Take(featureFlag, &model.FeatureFlag{UserID: userID}).Error; err != nil {
		return nil, convertError(err)
	}
	return featureFlag, nil
}