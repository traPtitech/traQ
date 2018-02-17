package model

import (
	"fmt"

	"github.com/satori/go.uuid"
)

//Device 通知デバイスの構造体
type Device struct {
	Token  string `xorm:"varchar(190) pk not null"`
	UserID string `xorm:"char(36) not null index"`
}

//TableName Device構造体のテーブル名
func (*Device) TableName() string {
	return "devices"
}

//Register デバイスを登録
func (device *Device) Register() error {
	if device.UserID == "" {
		return fmt.Errorf("UserID is empty")
	}

	if device.Token == "" {
		return fmt.Errorf("token is empty")
	}

	if _, err := db.Insert(device); err != nil {
		return fmt.Errorf("failed to create device: %v", err) //TODO すでに登録されていた場合に除く
	}

	return nil
}

// Unregister デバイスの登録を解除
func (device *Device) Unregister() error {
	if device.UserID == "" && device.Token == "" {
		return fmt.Errorf("both UserID and Token are empty")
	}

	if _, err := db.Delete(device); err != nil {
		return fmt.Errorf("failed to delete device: %v", err)
	}
	return nil
}

// GetDevices 指定ユーザーのデバイスを取得
func GetDevices(user uuid.UUID) ([]*Device, error) {
	var result []*Device
	if err := db.Find(&result, &Device{UserID: user.String()}); err != nil {
		return nil, fmt.Errorf("failed to get devices : %v", err)
	}
	return result, nil
}

// GetAllDevices 全ユーザーの全デバイスを取得
func GetAllDevices() ([]*Device, error) {
	var result []*Device
	if err := db.Find(&result); err != nil {
		return nil, fmt.Errorf("failed to get devices : %v", err)
	}
	return result, nil
}

// GetAllDeviceIDs 全ユーザーの全デバイスIDを取得
func GetAllDeviceIDs() ([]string, error) {
	var result []string
	if err := db.Table(&Device{}).Cols("token").Find(&result); err != nil {
		return nil, fmt.Errorf("failed to get devices : %v", err)
	}
	return result, nil
}

// GetDeviceIDs 指定ユーザーの全デバイスIDを取得
func GetDeviceIDs(user uuid.UUID) ([]string, error) {
	var result []string
	if err := db.Table(&Device{}).Cols("token").Find(&result, &Device{UserID: user.String()}); err != nil {
		return nil, fmt.Errorf("failed to get devices : %v", err)
	}
	return result, nil
}
