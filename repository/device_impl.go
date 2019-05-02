package repository

import (
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/traPtitech/traQ/model"
)

// RegisterDevice implements DeviceRepository interface.
func (repo *GormRepository) RegisterDevice(userID uuid.UUID, token string) (*model.Device, error) {
	var d model.Device
	err := repo.transact(func(tx *gorm.DB) error {
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

// UnregisterDevice implements DeviceRepository interface.
func (repo *GormRepository) UnregisterDevice(token string) (err error) {
	if len(token) == 0 {
		return nil
	}
	return repo.db.Delete(&model.Device{Token: token}).Error
}

// GetDevicesByUserID implements DeviceRepository interface.
func (repo *GormRepository) GetDevicesByUserID(userID uuid.UUID) (result []*model.Device, err error) {
	result = make([]*model.Device, 0)
	if userID == uuid.Nil {
		return result, nil
	}
	return result, repo.db.Where(&model.Device{UserID: userID}).Find(&result).Error
}

// GetDeviceTokensByUserID implements DeviceRepository interface.
func (repo *GormRepository) GetDeviceTokensByUserID(userID uuid.UUID) (result []string, err error) {
	result = make([]string, 0)
	if userID == uuid.Nil {
		return result, nil
	}
	return result, repo.db.Model(&model.Device{}).Where(&model.Device{UserID: userID}).Pluck("token", &result).Error
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
