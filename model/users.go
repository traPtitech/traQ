package model

import (
	"crypto/rand"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/utils/icon"
	"github.com/traPtitech/traQ/utils/validator"
	"io"
	"strings"
	"time"

	"github.com/labstack/gommon/log"
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
	ID          string     `xorm:"char(36) pk"                 validate:"required,uuid"`
	Name        string     `xorm:"varchar(32) unique not null" validate:"required,name"`
	DisplayName string     `xorm:"varchar(64) not null"        validate:"max=64"`
	Email       string     `xorm:"text not null"               validate:"required,email"`
	Password    string     `xorm:"char(128) not null"          validate:"required,max=128"`
	Salt        string     `xorm:"char(128) not null"          validate:"required,max=128"`
	Icon        string     `xorm:"char(36) not null"`
	Status      int        `xorm:"tinyint not null"`
	Bot         bool       `xorm:"bool not null"`
	Role        string     `xorm:"text not null"               validate:"required"`
	TwitterID   string     `xorm:"varchar(15) not null"        validate:"twitterid"`
	LastOnline  *time.Time `xorm:"timestamp"`
	CreatedAt   time.Time  `xorm:"created not null"`
	UpdatedAt   time.Time  `xorm:"updated not null"`
}

// GetUID ユーザーIDを取得します
func (user *User) GetUID() uuid.UUID {
	return uuid.FromStringOrNil(user.ID)
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

// Create userをDBに入れる
func (user *User) Create() error {
	user.ID = CreateUUID()
	user.Status = 1 // TODO: 状態確認

	if err := user.Validate(); err != nil {
		return err
	}
	if _, err := db.Insert(user); err != nil {
		return fmt.Errorf("Failed to create user object: %v", err)
	}

	iconID, err := GenerateIcon(user.Name)
	if err != nil {
		log.Error(err)
		return err
	}
	user.Icon = iconID

	return user.Update()
}

// GetUser IDでユーザーの構造体を取得する
func GetUser(userID string) (*User, error) {
	user := &User{ID: userID}

	if has, err := db.Get(user); err != nil {
		return nil, err
	} else if !has {
		return nil, ErrNotFound
	}

	return user, nil
}

// GetUsers ユーザーの一覧の取得
func GetUsers() ([]*User, error) {
	var users []*User
	// TODO ユーザーの状態によってフィルタ
	if err := db.Find(&users); err != nil {
		return nil, err
	}
	return users, nil
}

// SetPassword パスワードの設定を行う Createより前に実行する
func (user *User) SetPassword(pass string) error {
	salt, err := generateSalt()
	if err != nil {
		return fmt.Errorf("an error occurred while generating salt: %v", err)
	}

	user.Salt = hex.EncodeToString(salt)
	user.Password = hex.EncodeToString(hashPassword(pass, salt))
	return nil
}

// Exists 存在するuserを取得します
func (user *User) Exists() (bool, error) {
	if user.Name == "" {
		return false, fmt.Errorf("UserName is empty")
	}
	return db.Get(user)
}

// Authorization 認証を行う
func (user *User) Authorization(pass string) error {
	if has, err := db.Get(user); err != nil {
		return err
	} else if !has {
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

	if subtle.ConstantTimeCompare(storedPassword, hashPassword(pass, salt)) != 1 {
		return ErrUserWrongIDOrPassword
	}
	return nil
}

// Update ユーザー情報をデータベースに適用
func (user *User) Update() error {
	if err := user.Validate(); err != nil {
		return err
	}
	if _, err := db.Id(user.ID).UseBool().Update(user); err != nil {
		return fmt.Errorf("Failed to update user: %v", err)
	}

	return nil
}

// UpdateIconID ユーザーのアイコンを更新する
func (user *User) UpdateIconID(ID string) error {
	user.Icon = ID
	return user.Update()
}

// UpdateDisplayName ユーザーの表示名を変更する
func (user *User) UpdateDisplayName(name string) error {
	user.DisplayName = name
	return user.Update()
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
	return IsUserOnline(user.ID)
}

// UpdateUserLastOnline ユーザーの最終オンライン日時を更新します
func UpdateUserLastOnline(id string, time time.Time) (err error) {
	_, err = db.ID(id).Update(&User{LastOnline: &time})
	return err
}

func hashPassword(pass string, salt []byte) []byte {
	return pbkdf2.Key([]byte(pass), salt, 65536, 64, sha512.New)[:]
}

func generateSalt() ([]byte, error) {
	salt := make([]byte, 64)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}
	return salt, nil
}

// GenerateIcon svgアイコンを生成してそのファイルIDを返します
func GenerateIcon(salt string) (string, error) {
	svg := strings.NewReader(icon.Generate(salt))

	file := &File{
		Name:      salt + ".svg",
		Size:      int64(svg.Len()),
		CreatorID: serverUser.ID,
	}
	if err := file.Create(svg); err != nil {
		return "", err
	}

	return file.ID, nil
}
