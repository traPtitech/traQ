package repository

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
)

const defaultNotifyCitation = false

// UpdateNotifyCitation implements UserSettingsRepository interface
func (repo *GormRepository) UpdateNotifyCitation(userID uuid.UUID, isEnable bool) error {
	if userID == uuid.Nil {
		return ErrNilID
	}

	var settings model.UserSettings

	changes := &model.UserSettings{
		UserID:         userID,
		NotifyCitation: isEnable,
	}

	if err := repo.db.First(&settings, "user_id=?", userID).Error; err != nil {
		err = convertError(err)
		if err == ErrNotFound {
			if err = repo.db.Create(changes).Error; err != nil {
				return err
			}
			return nil
		}
		return err
	}
	if err := repo.db.Model(&settings).Updates(changes).Error; err != nil {
		return convertError(err)
	}

	return nil
}

// GetNotifyCitation implements UserSettingsRepository interface
func (repo *GormRepository) GetNotifyCitation(userID uuid.UUID) (bool, error) {
	if userID == uuid.Nil {
		return defaultNotifyCitation, ErrNilID
	}

	var settings = model.UserSettings{}

	if err := repo.db.First(&settings, "user_id=?", userID).Error; err != nil {
		err = convertError(err)
		if err == ErrNotFound {
			return defaultNotifyCitation, nil
		}
		return defaultNotifyCitation, err
	}

	return settings.IsNotifyCitationEnabled(), nil
}

// GetUserSettings implements UserSettingsRepository interface
func (repo *GormRepository) GetUserSettings(userID uuid.UUID) (*model.UserSettings, error) {
	if userID == uuid.Nil {
		return nil, ErrNilID
	}
	var settings = model.UserSettings{}

	if err := repo.db.First(&settings, "user_id=?", userID).Error; err != nil {
		err = convertError(err)
		dus := &model.UserSettings{
			UserID:         userID,
			NotifyCitation: defaultNotifyCitation,
		}
		if err == ErrNotFound {
			return dus, nil
		}
		return dus, err
	}

	return &settings, nil
}
