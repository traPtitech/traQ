package repository

import (
	"database/sql"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/model"
)

// UpdateUserGroupNameArgs ユーザーグループ更新引数
type UpdateUserGroupNameArgs struct {
	Name        string
	Description sql.NullString
	AdminUserID uuid.NullUUID
}

// UserGroupRepository ユーザーグループリポジトリー
type UserGroupRepository interface {
	CreateUserGroup(name, description string, adminID uuid.UUID) (*model.UserGroup, error)
	UpdateUserGroup(id uuid.UUID, args UpdateUserGroupNameArgs) error
	DeleteUserGroup(id uuid.UUID) error
	GetUserGroup(id uuid.UUID) (*model.UserGroup, error)
	GetUserGroupByName(name string) (*model.UserGroup, error)
	GetUserBelongingGroups(userID uuid.UUID) ([]*model.UserGroup, error)
	GetAllUserGroups() ([]*model.UserGroup, error)
	AddUserToGroup(userID, groupID uuid.UUID) error
	RemoveUserFromGroup(userID, groupID uuid.UUID) error
	GetUserGroupMemberIDs(groupID uuid.UUID) ([]uuid.UUID, error)
}
