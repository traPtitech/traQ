package gorm

import (
	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
)

const defaultNotifyCitation = false

// UpdateNotifyCitation implements UserSettingsRepository interface
func (repo *Repository) UpdateNotifyCitation(userID uuid.UUID, isEnable bool) error {
	if userID == uuid.Nil {
		return repository.ErrNilID
	}

	var settings = model.UserSettings{}

	if err := repo.db.First(&settings, "user_id=?", userID).Error; err != nil {
		err = convertError(err)
		if err == repository.ErrNotFound {
			if err = repo.db.Create(&model.UserSettings{
				UserID:         userID,
				NotifyCitation: isEnable,
			}).Error; err != nil {
				return err
			}
			return nil
		}
		return err
	}
	if err := repo.db.Model(&settings).Updates(map[string]interface{}{
		"user_id":         userID,
		"notify_citation": isEnable,
	}).Error; err != nil {
		return convertError(err)
	}

	return nil
}

// GetNotifyCitation implements UserSettingsRepository interface
func (repo *Repository) GetNotifyCitation(userID uuid.UUID) (bool, error) {
	if userID == uuid.Nil {
		return defaultNotifyCitation, repository.ErrNilID
	}

	var settings = model.UserSettings{}

	if err := repo.db.First(&settings, "user_id=?", userID).Error; err != nil {
		err = convertError(err)
		if err == repository.ErrNotFound {
			return defaultNotifyCitation, nil
		}
		return defaultNotifyCitation, err
	}

	return settings.IsNotifyCitationEnabled(), nil
}

// GetUserSettings implements UserSettingsRepository interface
func (repo *Repository) GetUserSettings(userID uuid.UUID) (*model.UserSettings, error) {
	if userID == uuid.Nil {
		return nil, repository.ErrNilID
	}
	var settings = model.UserSettings{}

	if err := repo.db.First(&settings, "user_id=?", userID).Error; err != nil {
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
