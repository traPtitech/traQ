package repository

import "github.com/traPtitech/traQ/model"

type UserRoleRepository interface {
	// CreateUserRoles ユーザーロールを作成します
	//
	// 成功した場合、nilを返します。
	// DBによるエラーを返すことがあります。
	CreateUserRoles(roles ...*model.UserRole) error
	// GetAllUserRoles 全てのユーザーロールを返します
	//
	// 成功した場合、ユーザーロールの配列とnilを返します。
	// DBによるエラーを返すことがあります。
	GetAllUserRoles() ([]*model.UserRole, error)
}
