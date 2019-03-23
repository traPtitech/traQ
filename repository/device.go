package repository

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
)

// DeviceRepository FCMデバイスリポジトリ
type DeviceRepository interface {
	RegisterDevice(userID uuid.UUID, token string) (*model.Device, error)
	UnregisterDevice(token string) (err error)
	GetDevicesByUserID(user uuid.UUID) (result []*model.Device, err error)
	GetDeviceTokensByUserID(user uuid.UUID) (result []string, err error)
	GetAllDevices() (result []*model.Device, err error)
	GetAllDeviceTokens() (result []string, err error)
}
