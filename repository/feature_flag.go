package repository

import (
	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/model"
)

// FeatureFlagRepository FeatureFlagリポジトリ
type FeatureFlagRepository interface {
	// CreateUserFeatureFlag ユーザーのFeatureFlagを作成します。
	//
	// 成功した場合、nilを返します。
	// userIDがuuid.Nilの場合、ErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	CreateUserFeatureFlag(userID uuid.UUID, featureFlagsJSON string) error
	// UpdateUserFeatureFlag ユーザーのFeatureFlagを更新します。
	//
	// 成功した場合、nilを返します。
	// 指定したUserIDに対応するFeatureFlagが存在しない場合ErrNotFoundを返します。
	// userIDがuuid.Nilの場合、ErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	UpdateUserFeatureFlag(UserID uuid.UUID, featureFlagsJSON string) error
	// GetFeatureFlagByUserID 指定したユーザーのFeatureFlagを取得します。
	//
	// 成功した場合、FeatureFlagとnilを返します。
	// 指定したUserIDに対応するFeatureFlagが存在しない場合ErrNotFoundを返します。
	// userIDがuuid.Nilの場合、ErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	GetFeatureFlagByUserID(userID uuid.UUID) (*model.FeatureFlag, error)
}
