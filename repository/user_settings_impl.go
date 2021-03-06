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
