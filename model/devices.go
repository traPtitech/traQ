package model

import (
	"fmt"
	"github.com/satori/go.uuid"
)

// 通知デバイスの構造体
type Device struct {
	Token  string `xorm:"varchar(255) pk not null"`
	UserId string `xorm:"char(36) not null index"`
}

// Device構造体のテーブル名
func (*Device) TableName() string {
	return "devices"
}

// デバイスを登録
func (device *Device) Register() error {
	if device.UserId == "" {
		return fmt.Errorf("UserId is empty")
	}

	if device.Token == "" {
		return fmt.Errorf("token is empty")
	}

	if _, err := db.Insert(device); err != nil {
		return fmt.Errorf("failed to create device: %v", err) //TODO すでに登録されていた場合に除く
	}

	return nil
}

// デバイスの登録を解除
func (device *Device) Unregister() error {
	if device.UserId == "" && device.Token == "" {
		return fmt.Errorf("both UserId and Token is empty")
	}

	if _, err := db.Delete(device); err != nil {
		return fmt.Errorf("failed to delete device: %v", err)
	}
	return nil
}

// 指定ユーザーのデバイスを取得
func GetDevices(user uuid.UUID) ([]*Device, error) {
	var result []*Device
	if err := db.Find(&result, &Device{UserId: user.String()}); err != nil {
		return nil, fmt.Errorf("failed to get devices : %v", err)
	}
	return result, nil
}

// 全ユーザーの全デバイスを取得
func GetAllDevices() ([]*Device, error) {
	var result []*Device
	if err := db.Find(&result); err != nil {
		return nil, fmt.Errorf("failed to get devices : %v", err)
	}
	return result, nil
}

// 全ユーザーの全デバイスIDを取得
func GetAllDeviceIds() ([]string, error) {
	var result []string
	if err := db.Table(&Device{}).Cols("token").Find(&result); err != nil {
		return nil, fmt.Errorf("failed to get devices : %v", err)
	}
	return result, nil
}

// 指定ユーザーの全デバイスIDを取得
func GetDeviceIds(user uuid.UUID) ([]string, error) {
	var result []string
	if err := db.Table(&Device{}).Cols("token").Find(&result, &Device{UserId: user.String()}); err != nil {
		return nil, fmt.Errorf("failed to get devices : %v", err)
	}
	return result, nil
}
