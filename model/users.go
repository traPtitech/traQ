package model

import (
	"bytes"
	"crypto/rand"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"github.com/jinzhu/gorm"
	"github.com/mikespook/gorbac"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/utils"
	"github.com/traPtitech/traQ/utils/validator"
	"image/png"
	"io"
	"time"
	"unicode/utf8"

	"golang.org/x/crypto/pbkdf2"
)

var (
	// ErrUserBotTryLogin : ユーザーエラー botユーザーでログインを試みました。botユーザーはログインできません。
	ErrUserBotTryLogin = errors.New("bot user is not allowed to login")
	// ErrUserWrongIDOrPassword : ユーザーエラー IDかパスワードが間違っています。
	ErrUserWrongIDOrPassword = errors.New("password or id is wrong")
)

// User userの構造体
type User struct {
	ID          string `gorm:"type:char(36);primary_key" validate:"required,uuid"`
	Name        string `gorm:"type:varchar(32);unique"   validate:"required,name"`
	DisplayName string `gorm:"type:varchar(64)"          validate:"max=64"`
	Email       string `gorm:"type:text"                 validate:"required,email"`
	Password    string `gorm:"type:char(128)"            validate:"required,max=128"`
	Salt        string `gorm:"type:char(128)"            validate:"required,max=128"`
	Icon        string `gorm:"type:char(36)"`
	Status      int    `gorm:"type:tinyint"`
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
	return uuid.Must(uuid.FromString(user.ID))
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

// GetLastOnline ユーザーの最終オンライン日時を取得します
func (user *User) GetLastOnline() time.Time {
	i, ok := currentUserOnlineMap.Load(user.ID)
	if !ok {
		if user.LastOnline == nil {
			return time.Time{}
		}
		return *user.LastOnline
	}
	return i.(*userOnlineStatus).getTime()
}

// IsOnline ユーザーがオンラインかどうかを返します
func (user *User) IsOnline() bool {
	return IsUserOnline(user.GetUID())
}

// CreateUser ユーザーを作成します
func CreateUser(name, email, password string, role gorbac.Role) (*User, error) {
	salt := generateSalt()
	user := &User{
		ID:       CreateUUID(),
		Name:     name,
		Email:    email,
		Password: hex.EncodeToString(hashPassword(password, salt)),
		Salt:     hex.EncodeToString(salt),
		Status:   1, //TODO 状態管理
		Bot:      false,
		Role:     role.ID(),
	}

	if err := user.Validate(); err != nil {
		return nil, err
	}

	iconID, err := GenerateIcon(user.Name)
	if err != nil {
		return nil, err
	}
	user.Icon = iconID

	if err := db.Create(user).Error; err != nil {
		return nil, err
	}

	return user, nil
}

// GetUser IDでユーザーの構造体を取得する
func GetUser(userID uuid.UUID) (*User, error) {
	user := &User{}
	if err := db.Where(&User{ID: userID.String()}).Take(user).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return user, nil
}

// UserExists 指定したIDのユーザーが存在するかどうか
func UserExists(userID uuid.UUID) (bool, error) {
	c := 0
	if err := db.Model(User{}).Where(&User{ID: userID.String()}).Count(&c).Error; err != nil {
		return false, err
	}
	return c > 0, nil
}

// GetUserByName nameでユーザーを取得します
func GetUserByName(name string) (*User, error) {
	if len(name) == 0 {
		return nil, ErrNotFound
	}
	user := &User{}
	if err := db.Where(&User{Name: name}).Take(user).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return user, nil
}

// GetUsers ユーザーの一覧の取得
func GetUsers() (users []*User, err error) {
	err = db.Find(&users).Error
	return
}

// ChangeUserPassword ユーザーのパスワードを変更します
func ChangeUserPassword(userID uuid.UUID, password string) error {
	salt := generateSalt()
	return db.Model(&User{ID: userID.String()}).Updates(map[string]interface{}{
		"salt":     hex.EncodeToString(salt),
		"password": hex.EncodeToString(hashPassword(password, salt)),
	}).Error
}

// ChangeUserIcon ユーザーのアイコンを変更します
func ChangeUserIcon(userID, fileID uuid.UUID) error {
	return db.Model(&User{ID: userID.String()}).Update("icon", fileID.String()).Error
}

// ChangeUserDisplayName ユーザーの表示名を変更します
func ChangeUserDisplayName(userID uuid.UUID, displayName string) error {
	if utf8.RuneCountInString(displayName) > 64 {
		return errors.New("displayName must be <=64 characters")
	}
	return db.Model(&User{ID: userID.String()}).Update("display_name", displayName).Error
}

// ChangeUserTwitterID ユーザーのTwitterIDを変更します
func ChangeUserTwitterID(userID uuid.UUID, twitterID string) error {
	if err := validator.ValidateVar(twitterID, "twitterid"); err != nil {
		return err
	}
	return db.Model(&User{ID: userID.String()}).Update("twitter_id", twitterID).Error
}

// UpdateUserLastOnline ユーザーの最終オンライン日時を更新します
func UpdateUserLastOnline(id uuid.UUID, time time.Time) (err error) {
	return db.Model(&User{ID: id.String()}).Update("last_online", &time).Error
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

	if subtle.ConstantTimeCompare(storedPassword, hashPassword(password, salt)) != 1 {
		return ErrUserWrongIDOrPassword
	}
	return nil
}

func hashPassword(pass string, salt []byte) []byte {
	return pbkdf2.Key([]byte(pass), salt, 65536, 64, sha512.New)[:]
}

func generateSalt() []byte {
	salt := make([]byte, 64)
	io.ReadFull(rand.Reader, salt)
	return salt
}

// GenerateIcon pngアイコンを生成してそのファイルIDを返します
func GenerateIcon(salt string) (string, error) {
	img := utils.GenerateIcon(salt)
	b := &bytes.Buffer{}
	if err := png.Encode(b, img); err != nil {
		return "", err
	}

	file := &File{
		Name:      salt + ".png",
		Size:      int64(b.Len()),
		CreatorID: serverUser.ID,
	}
	if err := file.Create(b); err != nil {
		return "", err
	}

	return file.ID, nil
}
