package repository

import (
	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils/optional"
)

// UpdateUserGroupArgs ユーザーグループ更新引数
type UpdateUserGroupArgs struct {
	Name        optional.Of[string]
	Description optional.Of[string]
	Type        optional.Of[string]
	Icon        optional.Of[uuid.UUID]
}

// UserGroupRepository ユーザーグループリポジトリ
type UserGroupRepository interface {
	// CreateUserGroup ユーザーグループを作成します
	//
	// 成功した場合、ユーザーグループとnilを返します。
	// 既にNameが使われている場合、ErrAlreadyExistsを返します。
	// DBによるエラーを返すことがあります。
	CreateUserGroup(name, description, gType string, adminID, iconFileID uuid.UUID) (*model.UserGroup, error)
	// UpdateUserGroup 指定したユーザーグループを更新します
	//
	// 成功した場合、nilを返します。
	// 存在しないグループの場合、ErrNotFoundを返します。
	// 引数にuuid.Nilを指定した場合、ErrNilIDを返します。
	// 既にNameが使われている場合、ErrAlreadyExistsを返します。
	// DBによるエラーを返すことがあります。
	UpdateUserGroup(id uuid.UUID, args UpdateUserGroupArgs) error
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
	// 存在しないグループの場合、ErrNotFoundを返します。
	// 引数にuuid.Nilを指定した場合、ErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	AddUserToGroup(userID, groupID uuid.UUID, role string) error
	// AddUsersToGroup 指定したグループに指定した複数のユーザーを追加します
	//
	// 成功した、或いは既に追加されている場合、nilを返します。
	// 存在しないグループの場合、ErrNotFoundを返します。
	// 引数にuuid.Nilを指定した場合、ErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	AddUsersToGroup(users []model.UserGroupMember, groupID uuid.UUID) error
	// RemoveUserFromGroup 指定したグループから指定したユーザーを削除します
	//
	// 全員の追加が成功した、或いは既に追加されている場合、nilを返します。
	// 存在しないグループの場合、ErrNotFoundを返します。
	// 引数にuuid.Nilを指定した場合、ErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	RemoveUserFromGroup(userID, groupID uuid.UUID) error
	// AddUserToGroupAdmin 指定したグループの管理者に指定したユーザーを追加します
	//
	// 成功した、或いは既に追加されている場合、nilを返します。
	// 存在しないグループの場合、ErrNotFoundを返します。
	// 引数にuuid.Nilを指定した場合、ErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	AddUserToGroupAdmin(userID, groupID uuid.UUID) error
	// RemoveUserFromGroupAdmin 指定したグループの管理者から指定したユーザーを削除します
	//
	// 成功した、或いは既に居ない場合、nilを返します。
	// 存在しないグループの場合、ErrNotFoundを返します。
	// グループから管理者が居なくなる場合、ErrForbiddenを返します。
	// 引数にuuid.Nilを指定した場合、ErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	RemoveUserFromGroupAdmin(userID, groupID uuid.UUID) error
}
