package model

import (
	"crypto/rand"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"io"
	"time"

	"errors"
	"github.com/GeorgeMac/idicon/colour"
	"github.com/GeorgeMac/idicon/icon"
	"github.com/labstack/gommon/log"
	"golang.org/x/crypto/pbkdf2"
	"regexp"
	"strings"
)

var (
	userNameRegex = regexp.MustCompile("^[a-zA-Z0-9_]{1,32}$")
	emailRegex    = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
)

// User userの構造体
type User struct {
	ID        string    `xorm:"char(36) pk"`
	Name      string    `xorm:"varchar(32) unique not null"`
	Email     string    `xorm:"text not null"`
	Password  string    `xorm:"char(128) not null"`
	Salt      string    `xorm:"char(128) not null"`
	Icon      string    `xorm:"char(36) not null"`
	Status    int       `xorm:"tinyint not null"`
	CreatedAt time.Time `xorm:"created not null"`
	UpdatedAt time.Time `xorm:"updated not null"`
}

// TableName dbの名前を指定する
func (user *User) TableName() string {
	return "users"
}

// Create userをDBに入れる
func (user *User) Create() error {
	if !userNameRegex.MatchString(user.Name) {
		return fmt.Errorf("invalid name")
	}

	if !emailRegex.MatchString(user.Email) {
		return fmt.Errorf("invalid email")
	}

	if user.Password == "" {
		return fmt.Errorf("password is empty")
	}

	if user.Salt == "" {
		return fmt.Errorf("salt is empty")
	}

	user.ID = CreateUUID()
	user.Status = 1 // TODO: 状態確認

	iconID, err := generateIcon(user.Name, user.ID)
	if err != nil {
		log.Error(err)
		return err
	}
	user.Icon = iconID

	if _, err := db.Insert(user); err != nil {
		return fmt.Errorf("Failed to create user object: %v", err)
	}
	return nil
}

// GetUser IDでユーザーの構造体を取得する
func GetUser(userID string) (*User, error) {
	var user = &User{}
	has, err := db.ID(userID).Get(user)

	if err != nil {
		return nil, fmt.Errorf("Failed to find user: %v", err)
	}
	if !has {
		return nil, fmt.Errorf("This userID doesn't exist: userID = %v", userID)
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
	user.Salt = salt
	user.Password = hashPassword(pass, user.Salt)

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
	if user.Name == "" {
		return fmt.Errorf("name is empty")
	}

	has, err := db.Get(user)
	if err != nil {
		return fmt.Errorf("Failed to find message: %v", err)
	}
	if !has {
		user.Salt, err = generateSalt()
		if err != nil {
			return fmt.Errorf("an error occurred while generating salt: %v", err)
		}
	}

	hashedPassword := hashPassword(pass, user.Salt)

	if subtle.ConstantTimeCompare([]byte(hashedPassword), []byte(user.Password)) != 1 {
		return fmt.Errorf("password or id is wrong")
	}
	return nil
}

// UpdateIconID ユーザーのアイコンを更新する
func (user *User) UpdateIconID(ID string) error {
	if len(user.ID) == 0 {
		return errors.New("invalid user")
	}
	user.Icon = ID
	_, err := db.ID(user.ID).UseBool().Update(user)
	return err
}

func hashPassword(pass, salt string) string {
	converted := pbkdf2.Key([]byte(pass), []byte(salt), 65536, 64, sha512.New)
	return hex.EncodeToString(converted[:])
}

func generateSalt() (string, error) {
	b := make([]byte, 14)

	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
}

func generateIcon(salt, userID string) (string, error) {
	props := icon.DefaultProps()
	props.BaseColour = colour.NewColour(0xf2, 0xf2, 0xf2)

	generator, err := icon.NewGenerator(5, 5, icon.With(props))
	if err != nil {
		return "", err
	}
	svg := strings.NewReader(generator.Generate([]byte(salt)).String())

	file := &File{
		Name:      salt + ".svg",
		Size:      int64(svg.Len()),
		CreatorID: userID,
	}
	if err := file.Create(svg); err != nil {
		return "", err
	}

	return file.ID, nil
}
