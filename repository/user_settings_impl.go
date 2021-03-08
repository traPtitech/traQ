package repository

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
)

// UpdateNotifyCitation implements UserSettingRepository interface
func (repo *GormRepository) UpdateNotifyCitation(userID uuid.UUID, isEnable bool) error {
	if userID == uuid.Nil {
		return ErrNilID
	}

	var settings model.UserSettings

	changes := map[string]interface{}{
		"user_id":        userID,
		"NotifyCitation": isEnable,
	}

	if err := repo.db.Model(&settings).Updates(changes).Error; err != nil {
		return convertError(err)
	}

	return nil
}

// GetNotifyCitation implements UserSettingRepository interface
func (repo *GormRepository) GetNotifyCitation(userID uuid.UUID) (bool, error) {
	if userID == uuid.Nil {
		return false, ErrNilID
	}

	var settings = &model.UserSettings{}

	if err := repo.db.First(&settings, "user_id=?", userID).Error; err != nil {
		return false, convertError(err)
	}

	return settings.IsNotifyCitationEnabled(), nil
}

// GetUserSettings implements UserSettingRepository interface
func (repo *GormRepository) GetUserSettings(userID uuid.UUID) (*model.UserSettings, error) {
	if userID == uuid.Nil {
		return nil, ErrNilID
	}

	var settings = &model.UserSettings{}

	if err := repo.db.First(&settings, "user_id=?", userID).Error; err != nil {
		return nil, convertError(err)
	}

	return settings, nil
}
