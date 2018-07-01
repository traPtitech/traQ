package model

import (
	"github.com/jinzhu/gorm"
	"github.com/traPtitech/traQ/utils/validator"
	"time"

	"github.com/satori/go.uuid"
)

//Device 通知デバイスの構造体
type Device struct {
	Token     string    `gorm:"size:190;primary_key"`
	UserID    string    `gorm:"type:char(36);index"`
	CreatedAt time.Time `gorm:"precision:6"`
}

//TableName Device構造体のテーブル名
func (*Device) TableName() string {
	return "devices"
}

// Validate 構造体を検証します
func (d *Device) Validate() error {
	return validator.ValidateStruct(d)
}

// RegisterDevice FCMデバイスを登録
func RegisterDevice(userID uuid.UUID, token string) (*Device, error) {
	d := &Device{}
	if err := db.Where(Device{
		Token: token,
	}).Take(&d).Error; err == nil {
		return d, nil
	} else if !gorm.IsRecordNotFoundError(err) {
		return nil, err
	}

	d = &Device{
		Token:  token,
		UserID: userID.String(),
	}

	err := db.Create(d).Error
	if err != nil {
		return nil, err
	}
	return d, nil
}

// UnregisterDevice FCMデバイスを解放
func UnregisterDevice(userID uuid.UUID, token string) (err error) {
	err = db.Where(Device{Token: token, UserID: userID.String()}).Delete(Device{}).Error
	return
}

// GetDevices 指定ユーザーのデバイスを取得
func GetDevices(user uuid.UUID) (result []*Device, err error) {
	err = db.Where(Device{UserID: user.String()}).Find(&result).Error
	return
}

// GetAllDevices 全ユーザーの全デバイスを取得
func GetAllDevices() (result []*Device, err error) {
	err = db.Find(&result).Error
	return
}

// GetAllDeviceIDs 全ユーザーの全デバイスIDを取得
func GetAllDeviceIDs() (result []string, err error) {
	err = db.Model(Device{}).Pluck("token", &result).Error
	return
}

// GetDeviceIDs 指定ユーザーの全デバイスIDを取得
func GetDeviceIDs(user uuid.UUID) (result []string, err error) {
	err = db.Where(Device{UserID: user.String()}).Pluck("token", &result).Error
	return
}
