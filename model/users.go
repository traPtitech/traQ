package model

import (
	"crypto/subtle"
	"encoding/hex"
	"errors"
	vd "github.com/go-ozzo/ozzo-validation"
	"github.com/gofrs/uuid"
	"github.com/spf13/viper"
	"github.com/traPtitech/traQ/utils"
	"github.com/traPtitech/traQ/utils/validator"
	"gopkg.in/guregu/null.v3"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	// ErrUserBotTryLogin : ユーザーエラー botユーザーでログインを試みました。botユーザーはログインできません。
	ErrUserBotTryLogin = errors.New("bot user is not allowed to login")
	// ErrUserWrongIDOrPassword : ユーザーエラー IDかパスワードが間違っています。
	ErrUserWrongIDOrPassword = errors.New("password or id is wrong")
)

// UserAccountStatus ユーザーアカウント状態
type UserAccountStatus int

// Valid 有効な値かどうか
func (v UserAccountStatus) Valid() bool {
	return userAccountStatuses[v]
}

// Int Int型にキャストします
func (v UserAccountStatus) Int() int {
	return int(v)
}

const (
	// UserAccountStatusDeactivated ユーザーアカウント状態: 凍結
	UserAccountStatusDeactivated UserAccountStatus = 0
	// UserAccountStatusActive ユーザーアカウント状態: 有効
	UserAccountStatusActive UserAccountStatus = 1
	// UserAccountStatusSuspended ユーザーアカウント状態: 一時停止
	UserAccountStatusSuspended UserAccountStatus = 2
)

var userAccountStatuses = map[UserAccountStatus]bool{
	UserAccountStatusDeactivated: true,
	UserAccountStatusActive:      true,
	UserAccountStatusSuspended:   true,
}

type UserType int

const (
	UserTypeHuman UserType = iota
	UserTypeBot
	UserTypeWebhook
)

type UserInfo interface {
	GetID() uuid.UUID
	GetName() string
	GetDisplayName() string
	GetIconFileID() uuid.UUID
	GetState() UserAccountStatus
	GetRole() string
	IsBot() bool
	GetCreatedAt() time.Time
	GetUpdatedAt() time.Time

	GetTwitterID() string
	GetBio() string
	GetLastOnline() null.Time

	// IsActive ユーザーが有効かどうか
	IsActive() bool
	GetResponseDisplayName() string
	GetUserType() UserType
	Authenticate(password string) error

	IsProfileAvailable() bool
}

// User userの構造体
type User struct {
	ID          uuid.UUID         `gorm:"type:char(36);not null;primary_key"`
	Name        string            `gorm:"type:varchar(32);not null;unique"`
	DisplayName string            `gorm:"type:varchar(64);not null;default:''"`
	Password    string            `gorm:"type:char(128);not null;default:''"`
	Salt        string            `gorm:"type:char(128);not null;default:''"`
	Icon        uuid.UUID         `gorm:"type:char(36);not null"`
	Status      UserAccountStatus `gorm:"type:tinyint;not null;default:0"`
	Bot         bool              `gorm:"type:boolean;not null;default:false"`
	Role        string            `gorm:"type:varchar(30);not null;default:'user'"`
	CreatedAt   time.Time         `gorm:"precision:6"`
	UpdatedAt   time.Time         `gorm:"precision:6"`

	Profile        *UserProfile            `gorm:"association_autoupdate:false;association_autocreate:false;preload:false;"`
	ExternalLogins []*ExternalProviderUser `gorm:"association_autoupdate:false;association_autocreate:false;preload:false;"`
}

// TableName dbの名前を指定する
func (user *User) TableName() string {
	return "users"
}

// Validate 構造体を検証します
func (user User) Validate() error {
	return vd.ValidateStruct(&user,
		vd.Field(&user.Name, validator.UserNameRuleRequired...),
		vd.Field(&user.DisplayName, vd.RuneLength(0, 64)),
		vd.Field(&user.Password, vd.Required, vd.RuneLength(128, 128)),
		vd.Field(&user.Salt, vd.Required, vd.RuneLength(128, 128)),
		vd.Field(&user.Role, vd.Required, vd.RuneLength(1, 30)),
	)
}

type UserProfile struct {
	UserID     uuid.UUID `gorm:"type:char(36);not null;primary_key"`
	Bio        string    `sql:"type:TEXT COLLATE utf8mb4_bin NOT NULL"`
	TwitterID  string    `gorm:"type:varchar(15);not null;default:''"`
	LastOnline null.Time `gorm:"precision:6"`
	UpdatedAt  time.Time `gorm:"precision:6"`
}

func (UserProfile) TableName() string {
	return "user_profiles"
}

type ExternalProviderUser struct {
	UserID       uuid.UUID `gorm:"type:char(36);not null;primary_key"`
	ProviderName string    `gorm:"type:varchar(30);not null;primary_key"`
	ExternalID   string    `gorm:"type:varchar(100);not null"`
	Extra        string    `gorm:"type:text;not null"`
	CreatedAt    time.Time `gorm:"precision:6"`
	UpdatedAt    time.Time `gorm:"precision:6"`
}

func (ExternalProviderUser) TableName() string {
	return "external_provider_users"
}

// GetID implements UserInfo interface
func (user *User) GetID() uuid.UUID {
	return user.ID
}

// GetName implements UserInfo interface
func (user *User) GetName() string {
	return user.Name
}

// GetDisplayName implements UserInfo interface
func (user *User) GetDisplayName() string {
	return user.DisplayName
}

// GetIconFileID implements UserInfo interface
func (user *User) GetIconFileID() uuid.UUID {
	return user.Icon
}

// GetState implements UserInfo interface
func (user *User) GetState() UserAccountStatus {
	return user.Status
}

// GetRole implements UserInfo interface
func (user *User) GetRole() string {
	return user.Role
}

// IsBot implements UserInfo interface
func (user *User) IsBot() bool {
	return user.Bot
}

// GetCreatedAt implements UserInfo interface
func (user *User) GetCreatedAt() time.Time {
	return user.CreatedAt
}

// GetUpdatedAt implements UserInfo interface
func (user *User) GetUpdatedAt() time.Time {
	if user.Profile != nil {
		if user.Profile.UpdatedAt.After(user.UpdatedAt) {
			return user.Profile.UpdatedAt
		}
	}
	return user.UpdatedAt
}

// GetTwitterID implements UserInfo interface
func (user *User) GetTwitterID() string {
	if user.Profile == nil {
		panic("unexpected control flow")
	}
	return user.Profile.TwitterID
}

// GetBio implements UserInfo interface
func (user *User) GetBio() string {
	if user.Profile == nil {
		panic("unexpected control flow")
	}
	return user.Profile.Bio
}

// GetLastOnline implements UserInfo interface
func (user *User) GetLastOnline() null.Time {
	if user.Profile == nil {
		panic("unexpected control flow")
	}
	return user.Profile.LastOnline
}

// IsActive implements UserInfo interface
func (user *User) IsActive() bool {
	return user.GetState() == UserAccountStatusActive
}

// GetResponseDisplayName implements UserInfo interface
func (user *User) GetResponseDisplayName() string {
	if len(user.GetDisplayName()) == 0 {
		return user.GetName()
	}
	return user.GetDisplayName()
}

// GetUserType implements UserInfo interface
func (user *User) GetUserType() UserType {
	if user.IsBot() {
		if strings.HasPrefix(user.GetName(), "Webhook") {
			return UserTypeWebhook
		}
		return UserTypeBot
	}
	return UserTypeHuman
}

// Authenticate implements UserInfo interface
func (user *User) Authenticate(password string) error {
	// Botはログイン不可
	if user.IsBot() {
		return ErrUserBotTryLogin
	}

	if viper.GetBool("externalAuthentication.enabled") {
		values := url.Values{}
		values.Set(viper.GetString("externalAuthentication.authPost.formUserNameKey"), user.GetName())
		values.Set(viper.GetString("externalAuthentication.authPost.formPasswordKey"), password)
		resp, err := http.PostForm(viper.GetString("externalAuthentication.authPost.url"), values)
		if err != nil {
			return ErrUserWrongIDOrPassword
		}
		defer resp.Body.Close()
		if resp.StatusCode != viper.GetInt("externalAuthentication.authPost.successfulCode") {
			return ErrUserWrongIDOrPassword
		}
	} else {
		if len(user.Password) == 0 || len(user.Salt) == 0 {
			return ErrUserWrongIDOrPassword
		}

		storedPassword, err := hex.DecodeString(user.Password)
		if err != nil {
			return ErrUserWrongIDOrPassword
		}
		salt, err := hex.DecodeString(user.Salt)
		if err != nil {
			return ErrUserWrongIDOrPassword
		}

		if subtle.ConstantTimeCompare(storedPassword, utils.HashPassword(password, salt)) != 1 {
			return ErrUserWrongIDOrPassword
		}
	}
	return nil
}

// IsProfileAvailable implements UserInfo interface
func (user *User) IsProfileAvailable() bool {
	return user.Profile != nil
}
