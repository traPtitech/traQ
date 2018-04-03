package model

import (
	"github.com/traPtitech/traQ/utils/validator"

	"github.com/satori/go.uuid"
)

//Device 通知デバイスの構造体
type Device struct {
	Token  string `xorm:"varchar(190) pk not null" validate:"required"`
	UserID string `xorm:"char(36) not null index"  validate:"uuid,required"`
}

//TableName Device構造体のテーブル名
func (*Device) TableName() string {
	return "devices"
}

// Validate 構造体を検証します
func (device *Device) Validate() error {
	return validator.ValidateStruct(device)
}

//Register デバイスを登録
func (device *Device) Register() error {
	if err := device.Validate(); err != nil {
		return err
	}

	var userID string

	if ok, err := db.Table(&Device{}).Where("token = ?", device.Token).Cols("user_id").Get(&userID); err != nil {
		return err
	} else if ok && userID == device.UserID {
		return nil
	}

	_, err := db.InsertOne(device)
	return err
}

// Unregister デバイスの登録を解除
func (device *Device) Unregister() (err error) {
	_, err = db.Delete(device)
	return
}

// GetDevices 指定ユーザーのデバイスを取得
func GetDevices(user uuid.UUID) (result []*Device, err error) {
	err = db.Find(&result, &Device{UserID: user.String()})
	return
}

// GetAllDevices 全ユーザーの全デバイスを取得
func GetAllDevices() (result []*Device, err error) {
	err = db.Find(&result)
	return
}

// GetAllDeviceIDs 全ユーザーの全デバイスIDを取得
func GetAllDeviceIDs() (result []string, err error) {
	err = db.Table(&Device{}).Cols("token").Find(&result)
	return
}

// GetDeviceIDs 指定ユーザーの全デバイスIDを取得
func GetDeviceIDs(user uuid.UUID) (result []string, err error) {
	err = db.Table(&Device{}).Cols("token").Find(&result, &Device{UserID: user.String()})
	return
}
