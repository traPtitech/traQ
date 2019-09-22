package repository

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils/set"
)

// DeviceRepository FCMデバイスリポジトリ
type DeviceRepository interface {
	// RegisterDevice FCMデバイスを登録します
	//
	// 成功した、或いは既に登録されていた場合に*model.Deviceとnilを返します。
	// 引数にuuid.Nilを指定した場合、ErrNilIDを返します。
	// tokenが空文字列の場合、ArgumentErrorを返します。
	// 登録しようとしたトークンが既に他のユーザーと関連づけられていた場合はArgumentErrorを返します。
	// DBによるエラーを返すことがあります。
	RegisterDevice(userID uuid.UUID, token string) (*model.Device, error)
	// DeleteDeviceTokens FCMデバイスの登録を解除します
	//
	// 成功した、或いは既に登録解除されていた場合にnilを返します。
	// DBによるエラーを返すことがあります。
	DeleteDeviceTokens(tokens []string) error
	// GetDevicesByUserID 指定したユーザーのデバイスを全て取得します
	//
	// 成功した場合、*model.Deviceの配列とnilを返します。
	// 存在しないユーザーを指定した場合は空配列とnilを返します。
	// DBによるエラーを返すことがあります。
	GetDevicesByUserID(userID uuid.UUID) (result []*model.Device, err error)
	// GetDeviceTokens 指定したユーザーの全デバイストークンを取得します
	//
	// 成功した場合、デバイストークンの配列とnilを返します。
	// DBによるエラーを返すことがあります。
	GetDeviceTokens(userIDs set.UUIDSet) ([]string, error)
	// GetAllDevices 全ユーザーの全デバイスを取得します
	//
	// DBによるエラーを返すことがあります。
	GetAllDevices() (result []*model.Device, err error)
	// GetAllDeviceIDs 全ユーザーの全デバイストークンを取得します
	//
	// DBによるエラーを返すことがあります。
	GetAllDeviceTokens() (result []string, err error)
}
