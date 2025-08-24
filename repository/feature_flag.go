package repository

import (
	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/model"
)

// FeatureFlagRepository FeatureFlagリポジトリ
type FeatureFlagRepository interface {
	// GetFeatureFlagByUserID 指定したユーザーのFeatureFlagを取得します。
	//
	// 成功した場合、FeatureFlagとnilを返します。
	// 存在しないFeatureFlagの場合ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	GetFeatureFlagByUserID(userID uuid.UUID) (*model.FeatureFlag, error)
}