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

// GetDevicesByUserID implements DeviceRepository interface.
func (repo *GormRepository) GetDevicesByUserID(userID uuid.UUID) (result []*model.Device, err error) {
	result = make([]*model.Device, 0)
	if userID == uuid.Nil {
		return result, nil
	}
	return result, repo.db.Where(&model.Device{UserID: userID}).Find(&result).Error
}

// GetAllDevices implements DeviceRepository interface.
func (repo *GormRepository) GetAllDevices() (result []*model.Device, err error) {
	result = make([]*model.Device, 0)
	return result, repo.db.Find(&result).Error
}

// GetAllDeviceIDs implements DeviceRepository interface.
func (repo *GormRepository) GetAllDeviceTokens() (result []string, err error) {
	result = make([]string, 0)
	return result, repo.db.Model(&model.Device{}).Pluck("token", &result).Error
}

// GetDeviceTokens implements DeviceRepository interface.
func (repo *GormRepository) GetDeviceTokens(userIDs set.UUIDSet) (tokens []string, err error) {
	tokens = make([]string, 0)
	return tokens, repo.db.Model(&model.Device{}).Where("user_id IN (?)", userIDs.StringArray()).Pluck("token", &tokens).Error
}

// DeleteDeviceTokens implements DeviceRepository interface.
func (repo *GormRepository) DeleteDeviceTokens(tokens []string) error {
	return repo.db.Where("token IN (?)", tokens).Delete(&model.Device{}).Error
}
