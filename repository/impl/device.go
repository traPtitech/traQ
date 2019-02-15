package impl

import (
	"errors"
	"github.com/jinzhu/gorm"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/model"
)

// RegisterDevice FCMデバイスを登録
func (repo *RepositoryImpl) RegisterDevice(userID uuid.UUID, token string) (*model.Device, error) {
	d := &model.Device{}
	if err := repo.db.Where(&model.Device{Token: token}).Take(&d).Error; err == nil {
		if d.UserID != userID {
			return nil, errors.New("the token has already been associated with other user")
		}
		return d, nil
	} else if !gorm.IsRecordNotFoundError(err) {
		return nil, err
	}

	d = &model.Device{
		Token:  token,
		UserID: userID,
	}
	return d, repo.db.Create(d).Error
}

// UnregisterDevice FCMデバイスを削除
func (repo *RepositoryImpl) UnregisterDevice(token string) (err error) {
	if len(token) == 0 {
		return nil
	}
	return repo.db.Delete(&model.Device{Token: token}).Error
}

// GetDevicesByUserID ユーザーのデバイスを取得
func (repo *RepositoryImpl) GetDevicesByUserID(user uuid.UUID) (result []*model.Device, err error) {
	if user == uuid.Nil {
		return nil, nil
	}
	err = repo.db.Where(&model.Device{UserID: user}).Find(&result).Error
	return result, err
}

// GetDeviceTokensByUserID ユーザーの全デバイストークンを取得
func (repo *RepositoryImpl) GetDeviceTokensByUserID(user uuid.UUID) (result []string, err error) {
	if user == uuid.Nil {
		return nil, nil
	}
	err = repo.db.Model(&model.Device{}).Where(&model.Device{UserID: user}).Pluck("token", &result).Error
	return result, err
}

// GetAllDevices 全ユーザーの全デバイスを取得
func (repo *RepositoryImpl) GetAllDevices() (result []*model.Device, err error) {
	err = repo.db.Find(&result).Error
	return result, err
}

// GetAllDeviceIDs 全ユーザーの全デバイストークンを取得
func (repo *RepositoryImpl) GetAllDeviceTokens() (result []string, err error) {
	err = repo.db.Model(&model.Device{}).Pluck("token", &result).Error
	return result, err
}
