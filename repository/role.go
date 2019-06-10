package repository

import (
	"github.com/traPtitech/traQ/model"
)

// UserRoleRepository ユーザー定義ロールリポジトリ
type UserRoleRepository interface {
	// GetAllRoles 全てのユーザーロールを取得します
	//
	// DBによるエラーを返すことがあります。
	GetAllRoles() ([]*model.UserRole, error)
}
