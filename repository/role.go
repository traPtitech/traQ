package repository

import (
	"github.com/traPtitech/traQ/model"
	"gopkg.in/guregu/null.v3"
)

// UpdateRoleArgs ユーザーロール更新引数
type UpdateRoleArgs struct {
	Permissions  []string
	Inheritances []string
	OAuth2Scope  null.Bool
}

// UserRoleRepository ユーザーロールリポジトリ
type UserRoleRepository interface {
	// GetAllRoles 全てのユーザーロールを取得します
	//
	// DBによるエラーを返すことがあります。
	GetAllRoles() ([]*model.UserRole, error)
	// GetRole 指定した名前のユーザーロールを取得します
	//
	// 存在しなかった場合、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	GetRole(role string) (*model.UserRole, error)
	// CreateRole ユーザーロールを作成します
	//
	// 引数に問題がある場合、ArgumentErrorを返します。
	// 既にNameが使われている場合、ErrAlreadyExistsを返します。
	// DBによるエラーを返すことがあります。
	CreateRole(name string) error
	// UpdateRole 指定したユーザーロールを更新します
	//
	// 引数に問題がある場合、ArgumentErrorを返します。
	// 存在しないロールの場合、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	UpdateRole(role string, args UpdateRoleArgs) error
}
