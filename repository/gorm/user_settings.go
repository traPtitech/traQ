package gorm

import (
	"context"

	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
)

const defaultNotifyCitation = false

// UpdateNotifyCitation implements UserSettingsRepository interface
func (repo *Repository) UpdateNotifyCitation(ctx context.Context, userID uuid.UUID, isEnable bool) error {
	if userID == uuid.Nil {
		return repository.ErrNilID
	}

	settings := model.UserSettings{}

	if err := repo.db.WithContext(ctx).First(&settings, "user_id=?", userID).Error; err != nil {
		err = convertError(err)
		if err == repository.ErrNotFound {
			return repo.db.WithContext(ctx).Create(&model.UserSettings{
				UserID:         userID,
				NotifyCitation: isEnable,
			}).Error
		}
		return err
	}
	if err := repo.db.WithContext(ctx).Model(&settings).Updates(map[string]interface{}{
		"user_id":         userID,
		"notify_citation": isEnable,
	}).Error; err != nil {
		return convertError(err)
	}

	return nil
}

// GetNotifyCitation implements UserSettingsRepository interface
func (repo *Repository) GetNotifyCitation(ctx context.Context, userID uuid.UUID) (bool, error) {
	if userID == uuid.Nil {
		return defaultNotifyCitation, repository.ErrNilID
	}

	settings := model.UserSettings{}

	if err := repo.db.WithContext(ctx).First(&settings, "user_id=?", userID).Error; err != nil {
		err = convertError(err)
		if err == repository.ErrNotFound {
			return defaultNotifyCitation, nil
		}
		return defaultNotifyCitation, err
	}

	return settings.IsNotifyCitationEnabled(), nil
}

// GetUserSettings implements UserSettingsRepository interface
func (repo *Repository) GetUserSettings(ctx context.Context, userID uuid.UUID) (*model.UserSettings, error) {
	if userID == uuid.Nil {
		return nil, repository.ErrNilID
	}
	settings := model.UserSettings{}

	if err := repo.db.WithContext(ctx).First(&settings, "user_id=?", userID).Error; err != nil {
		err = convertError(err)
		dus := &model.UserSettings{
			UserID:         userID,
			NotifyCitation: defaultNotifyCitation,
		}
		if err == repository.ErrNotFound {
			return dus, nil
		}
		return dus, err
	}

	return &settings, nil
}
