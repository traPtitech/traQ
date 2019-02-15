package repository

import (
	"github.com/mikespook/gorbac"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/model"
	"time"
)

// UserRepository ユーザーリポジトリ
type UserRepository interface {
	CreateUser(name, email, password string, role gorbac.Role) (*model.User, error)
	GetUser(id uuid.UUID) (*model.User, error)
	GetUserByName(name string) (*model.User, error)
	GetUsers() ([]*model.User, error)
	UserExists(id uuid.UUID) (bool, error)
	ChangeUserDisplayName(id uuid.UUID, displayName string) error
	ChangeUserPassword(id uuid.UUID, password string) error
	ChangeUserIcon(id, fileID uuid.UUID) error
	ChangeUserTwitterID(id uuid.UUID, twitterID string) error
	UpdateUserLastOnline(id uuid.UUID, time time.Time) (err error)
	IsUserOnline(id uuid.UUID) bool
	GetUserLastOnline(id uuid.UUID) (time.Time, error)
	GetHeartbeatStatus(channelID uuid.UUID) (model.HeartbeatStatus, bool)
	UpdateHeartbeatStatus(userID, channelID uuid.UUID, status string)
}
