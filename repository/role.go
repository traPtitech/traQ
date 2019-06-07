package repository

import "github.com/traPtitech/traQ/model"

// UserDefinedRoleRepository ユーザー定義ロールリポジトリ
type UserDefinedRoleRepository interface {
	// GetAllRoles 全てのユーザー定義ロールを取得します
	//
	// DBによるエラーを返すことがあります。
	GetAllRoles() ([]*model.UserDefinedRole, error)
}
