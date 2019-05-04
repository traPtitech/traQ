package repository

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
	"gopkg.in/guregu/null.v3"
)

// UpdateUserGroupNameArgs ユーザーグループ更新引数
type UpdateUserGroupNameArgs struct {
	Name        null.String
	Description null.String
	AdminUserID uuid.NullUUID
	Type        null.String
}

// UserGroupRepository ユーザーグループリポジトリ
type UserGroupRepository interface {
	// CreateUserGroup ユーザーグループを作成します
	//
	// 成功した場合、ユーザーグループとnilを返します。
	// 引数に問題がある場合、ArgumentErrorを返します。
	// 既にNameが使われている場合、ErrAlreadyExistsを返します。
	// DBによるエラーを返すことがあります。
	CreateUserGroup(name, description, gType string, adminID uuid.UUID) (*model.UserGroup, error)
	// UpdateUserGroup 指定したユーザーグループを更新します
	//
	// 成功した場合、nilを返します。
	// 存在しないグループの場合、ErrNotFoundを返します。
	// 引数に問題がある場合、ArgumentErrorを返します。
	// 引数にuuid.Nilを指定した場合、ErrNilIDを返します。
	// 既にNameが使われている場合、ErrAlreadyExistsを返します。
	// DBによるエラーを返すことがあります。
	UpdateUserGroup(id uuid.UUID, args UpdateUserGroupNameArgs) error
	// DeleteUserGroup 指定したユーザーグループを削除します
	//
	// 成功した場合、nilを返します。
	// 引数にuuid.Nilを指定した場合、ErrNilIDを返します。
	// 存在しないグループの場合、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	DeleteUserGroup(id uuid.UUID) error
	// GetUserGroup 指定したIDのユーザーグループを取得します
	//
	// 成功した場合、ユーザーグループとnilを返します。
	// 存在しないグループの場合、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	GetUserGroup(id uuid.UUID) (*model.UserGroup, error)
	// GetUserGroupByName 指定した名前のユーザーグループを取得します
	//
	// 成功した場合、ユーザーグループとnilを返します。
	// 存在しないグループの場合、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	GetUserGroupByName(name string) (*model.UserGroup, error)
	// GetUserBelongingGroupIDs 指定したユーザーが所属しているグループのUUIDを取得します
	//
	// 成功した場合、ユーザーグループのUUIDの配列とnilを返します。
	// 存在しないユーザーを指定した場合は空配列とnilを返します。
	// DBによるエラーを返すことがあります。
	GetUserBelongingGroupIDs(userID uuid.UUID) ([]uuid.UUID, error)
	// GetAllUserGroups 全てのグループを取得します
	//
	// 成功した場合、ユーザーグループの配列とnilを返します。
	// DBによるエラーを返すことがあります。
	GetAllUserGroups() ([]*model.UserGroup, error)
	// AddUserToGroup 指定したグループに指定したユーザーを追加します
	//
	// 成功した、或いは既に追加されている場合、nilを返します。
	// 引数にuuid.Nilを指定した場合、ErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	AddUserToGroup(userID, groupID uuid.UUID) error
	// RemoveUserFromGroup 指定したグループから指定したユーザーを削除します
	//
	// 成功した、或いは既に居ない場合、nilを返します。
	// 引数にuuid.Nilを指定した場合、ErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	RemoveUserFromGroup(userID, groupID uuid.UUID) error
	// GetUserGroupMemberIDs 指定したグループのメンバーのUUIDを取得します
	//
	// 成功した場合、UUIDの配列とnilを返します。
	// 存在しないグループを指定した場合は空配列とnilを返します。
	// DBによるエラーを返すことがあります。
	GetUserGroupMemberIDs(groupID uuid.UUID) ([]uuid.UUID, error)
}
