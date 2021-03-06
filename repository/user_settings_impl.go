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
		return err
	}

	return nil
}

// GetNotifyCitation implements UserSettingRepository interface
func (repo *GormRepository) GetNotifyCitation(userID uuid.UUID) (*model.UserSettings, error) {
	if userID == uuid.Nil {
		return nil, ErrNilID
	}

	var settings = &model.UserSettings{}

	if err := repo.db.Find(&settings, "user_id=?", userID).Error; err != nil {
		return nil, err
	}

	return settings, nil
}
