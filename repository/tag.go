package repository

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
)

// TagRepository ユーザータグリポジトリ
type TagRepository interface {
	CreateTag(name string, restricted bool, tagType string) (*model.Tag, error)
	ChangeTagType(id uuid.UUID, tagType string) error
	ChangeTagRestrict(id uuid.UUID, restrict bool) error
	GetAllTags() ([]*model.Tag, error)
	GetTagByID(id uuid.UUID) (*model.Tag, error)
	GetTagByName(name string) (*model.Tag, error)
	GetOrCreateTagByName(name string) (*model.Tag, error)
	AddUserTag(userID, tagID uuid.UUID) error
	ChangeUserTagLock(userID, tagID uuid.UUID, locked bool) error
	DeleteUserTag(userID, tagID uuid.UUID) error
	GetUserTag(userID, tagID uuid.UUID) (*model.UsersTag, error)
	GetUserTagsByUserID(userID uuid.UUID) ([]*model.UsersTag, error)
	GetUsersByTag(tag string) ([]*model.User, error)
	GetUserIDsByTag(tag string) ([]uuid.UUID, error)
	GetUserIDsByTagID(tagID uuid.UUID) ([]uuid.UUID, error)
}
