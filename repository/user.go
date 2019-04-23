package repository

import (
	"github.com/gofrs/uuid"
	"github.com/mikespook/gorbac"
	"github.com/traPtitech/traQ/model"
	"gopkg.in/guregu/null.v3"
	"time"
)

// UpdateUserArgs User情報更新引数
type UpdateUserArgs struct {
	DisplayName null.String
	TwitterID   null.String
	Role        null.String
}

// UserRepository ユーザーリポジトリ
type UserRepository interface {
	CreateUser(name, password string, role gorbac.Role) (*model.User, error)
	GetUser(id uuid.UUID) (*model.User, error)
	GetUserByName(name string) (*model.User, error)
	GetUsers() ([]*model.User, error)
	UserExists(id uuid.UUID) (bool, error)
	UpdateUser(id uuid.UUID, args UpdateUserArgs) error
	ChangeUserPassword(id uuid.UUID, password string) error
	ChangeUserIcon(id, fileID uuid.UUID) error
	ChangeUserAccountStatus(id uuid.UUID, status model.UserAccountStatus) error
	UpdateUserLastOnline(id uuid.UUID, time time.Time) (err error)
	IsUserOnline(id uuid.UUID) bool
	GetUserLastOnline(id uuid.UUID) (time.Time, error)
	GetHeartbeatStatus(channelID uuid.UUID) (model.HeartbeatStatus, bool)
	UpdateHeartbeatStatus(userID, channelID uuid.UUID, status string)
}
