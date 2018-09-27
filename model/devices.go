package model

import (
	"errors"
	"github.com/jinzhu/gorm"
	"time"

	"github.com/satori/go.uuid"
)

//Device 通知デバイスの構造体
type Device struct {
	Token     string    `gorm:"type:varchar(190);primary_key"`
	UserID    uuid.UUID `gorm:"type:char(36);index"`
	CreatedAt time.Time `gorm:"precision:6"`
}

//TableName Device構造体のテーブル名
func (*Device) TableName() string {
	return "devices"
}

// RegisterDevice FCMデバイスを登録
func RegisterDevice(userID uuid.UUID, token string) (*Device, error) {
	d := &Device{}
	if err := db.Where(&Device{
		Token: token,
	}).Take(&d).Error; err == nil {
		if d.UserID != userID {
			return nil, errors.New("the token has already been associated with other user")
		}
		return d, nil
	} else if !gorm.IsRecordNotFoundError(err) {
		return nil, err
	}

	d = &Device{
		Token:  token,
		UserID: userID,
	}

	err := db.Create(d).Error
	if err != nil {
		return nil, err
	}
	return d, nil
}

// UnregisterDevice FCMデバイスを解放
func UnregisterDevice(token string) (err error) {
	err = db.Delete(&Device{Token: token}).Error
	return
}

// GetDevices 指定ユーザーのデバイスを取得
func GetDevices(user uuid.UUID) (result []*Device, err error) {
	err = db.Where(&Device{UserID: user}).Find(&result).Error
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
	err = db.Model(Device{}).Where(&Device{UserID: user}).Pluck("token", &result).Error
	return
}
