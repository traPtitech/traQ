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

// UserGroupRepository ユーザーグループリポジトリー
type UserGroupRepository interface {
	CreateUserGroup(name, description string, adminID uuid.UUID) (*model.UserGroup, error)
	UpdateUserGroup(id uuid.UUID, args UpdateUserGroupNameArgs) error
	DeleteUserGroup(id uuid.UUID) error
	GetUserGroup(id uuid.UUID) (*model.UserGroup, error)
	GetUserGroupByName(name string) (*model.UserGroup, error)
	GetUserBelongingGroupIDs(userID uuid.UUID) ([]uuid.UUID, error)
	GetAllUserGroups() ([]*model.UserGroup, error)
	AddUserToGroup(userID, groupID uuid.UUID) error
	RemoveUserFromGroup(userID, groupID uuid.UUID) error
	GetUserGroupMemberIDs(groupID uuid.UUID) ([]uuid.UUID, error)
}
