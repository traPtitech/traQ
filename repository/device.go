package repository

import (
	"context"

	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils/optional"
	"github.com/traPtitech/traQ/utils/set"
)

type RegisterDeviceArgs struct {
	Token optional.Of[string]
	FID   optional.Of[string]
}
type TokenEntry struct {
	Token     string
	TokenType model.DeviceTokenType
}

// DeviceRepository FCMデバイスリポジトリ
type DeviceRepository interface {
	// RegisterDevice FCMデバイスを登録します
	//
	// 成功した、或いは既に登録されていた場合にnilを返します。
	// 引数にuuid.Nilを指定した場合、ErrNilIDを返します。
	// tokenが空文字列の場合、ArgumentErrorを返します。
	// 登録しようとしたトークンが既に他のユーザーと関連づけられていた場合はArgumentErrorを返します。
	// DBによるエラーを返すことがあります。
	RegisterDevice(ctx context.Context, userID uuid.UUID, args RegisterDeviceArgs) error
	// GetDeviceTokens 指定したユーザーの全デバイストークンを取得します
	//
	// 成功した場合、デバイストークンの配列とnilを返します。
	// DBによるエラーを返すことがあります。
	GetDeviceTokens(ctx context.Context, userIDs set.UUID) (map[uuid.UUID][]TokenEntry, error)
	// DeleteDeviceTokens FCMデバイスの登録を解除します
	//
	// 成功した、或いは既に登録解除されていた場合にnilを返します。
	// DBによるエラーを返すことがあります。
	DeleteDeviceTokens(ctx context.Context, args []RegisterDeviceArgs) error
}
