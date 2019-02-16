package model

import (
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/utils"
	"github.com/traPtitech/traQ/utils/validator"
	"time"
)

var (
	// ErrUserBotTryLogin : ユーザーエラー botユーザーでログインを試みました。botユーザーはログインできません。
	ErrUserBotTryLogin = errors.New("bot user is not allowed to login")
	// ErrUserWrongIDOrPassword : ユーザーエラー IDかパスワードが間違っています。
	ErrUserWrongIDOrPassword = errors.New("password or id is wrong")
)

// User userの構造体
type User struct {
	ID          uuid.UUID `gorm:"type:char(36);primary_key"`
	Name        string    `gorm:"type:varchar(32);unique"   validate:"required,name"`
	DisplayName string    `gorm:"type:varchar(64)"          validate:"max=64"`
	Email       string    `gorm:"type:text"                 validate:"required,email"`
	Password    string    `gorm:"type:char(128)"            validate:"required,max=128"`
	Salt        string    `gorm:"type:char(128)"            validate:"required,max=128"`
	Icon        uuid.UUID `gorm:"type:char(36)"`
	Status      int       `gorm:"type:tinyint"`
	Bot         bool
	Role        string     `gorm:"type:text"                 validate:"required"`
	TwitterID   string     `gorm:"type:varchar(15)"          validate:"twitterid"`
	LastOnline  *time.Time `gorm:"precision:6"`
	CreatedAt   time.Time  `gorm:"precision:6"`
	UpdatedAt   time.Time  `gorm:"precision:6"`
	DeletedAt   *time.Time `gorm:"precision:6"`
}

// GetUID ユーザーIDを取得します
func (user *User) GetUID() uuid.UUID {
	return user.ID
}

// GetName ユーザー名を取得します
func (user *User) GetName() string {
	return user.Name
}

// TableName dbの名前を指定する
func (user *User) TableName() string {
	return "users"
}

// Validate 構造体を検証します
func (user *User) Validate() error {
	return validator.ValidateStruct(user)
}

// AuthenticateUser ユーザー構造体とパスワードを照合します
func AuthenticateUser(user *User, password string) error {
	if user == nil {
		return ErrUserWrongIDOrPassword
	}
	// Botはログイン不可
	if user.Bot {
		return ErrUserBotTryLogin
	}

	storedPassword, err := hex.DecodeString(user.Password)
	if err != nil {
		return err
	}
	salt, err := hex.DecodeString(user.Salt)
	if err != nil {
		return err
	}

	if subtle.ConstantTimeCompare(storedPassword, utils.HashPassword(password, salt)) != 1 {
		return ErrUserWrongIDOrPassword
	}
	return nil
}

// UserStatus userの状態
type UserStatus struct {
	UserID   uuid.UUID `json:"userId"`
	Status   string    `json:"status"`
	LastTime time.Time `json:"-"`
}

// HeartbeatStatus Heartbeatの状態
type HeartbeatStatus struct {
	UserStatuses []*UserStatus `json:"userStatuses"`
	ChannelID    uuid.UUID     `json:"channelId"`
}
