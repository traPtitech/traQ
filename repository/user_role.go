//go:generate mockgen -source=$GOFILE -destination=mock_$GOPACKAGE/mock_$GOFILE
package repository

import (
	"context"

	"github.com/traPtitech/traQ/model"
)

type UserRoleRepository interface {
	// CreateUserRoles ユーザーロールを作成します
	//
	// 成功した場合、nilを返します。
	// DBによるエラーを返すことがあります。
	CreateUserRoles(ctx context.Context, roles ...*model.UserRole) error
	// GetAllUserRoles 全てのユーザーロールを返します
	//
	// 成功した場合、ユーザーロールの配列とnilを返します。
	// DBによるエラーを返すことがあります。
	GetAllUserRoles(ctx context.Context) ([]*model.UserRole, error)
}
