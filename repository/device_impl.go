package repository

import (
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils/set"
)

// RegisterDevice implements DeviceRepository interface.
func (repo *GormRepository) RegisterDevice(userID uuid.UUID, token string) (*model.Device, error) {
	if userID == uuid.Nil {
		return nil, ErrNilID
	}
	if len(token) == 0 {
		return nil, ArgError("Token", "token is empty")
	}

	var d model.Device
	err := repo.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Take(&d, &model.Device{Token: token}).Error; err == nil {
			if d.UserID != userID {
				return ArgError("Token", "the Token has already been associated with other user")
			}
			return nil
		} else if !gorm.IsRecordNotFoundError(err) {
			return err
		}

		d = model.Device{
			Token:  token,
			UserID: userID,
		}
		return tx.Create(&d).Error
	})
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// GetDeviceTokens implements DeviceRepository interface.
func (repo *GormRepository) GetDeviceTokens(userIDs set.UUIDSet) (tokens map[uuid.UUID][]string, err error) {
	var tmp []*model.Device
	if err := repo.db.Where("user_id IN (?)", userIDs.StringArray()).Find(&tmp).Error; err != nil {
		return nil, err
	}

	tokens = make(map[uuid.UUID][]string, len(userIDs))
	for _, device := range tmp {
		tokens[device.UserID] = append(tokens[device.UserID], device.Token)
	}
	return tokens, nil
}

// DeleteDeviceTokens implements DeviceRepository interface.
func (repo *GormRepository) DeleteDeviceTokens(tokens []string) error {
	return repo.db.Where("token IN (?)", tokens).Delete(&model.Device{}).Error
}
