package model

import (
	"crypto/rand"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"io"

	"golang.org/x/crypto/pbkdf2"
)

// User userの構造体
type User struct {
	ID        string `xorm:"char(36) pk"`
	Name      string `xorm:"varchar(32) unique not null"`
	Email     string `xorm:"text not null"`
	Password  string `xorm:"char(128) not null"`
	Salt      string `xorm:"char(128) not null"`
	Icon      string `xorm:"char(36) not null"`
	Status    int    `xorm:"tinyint not null"`
	CreatedAt string `xorm:"created not null"`
	UpdatedAt string `xorm:"updated not null"`
}

// TableName dbの名前を指定する
func (user *User) TableName() string {
	return "users"
}

// Create userをDBに入れる
func (user *User) Create() error {
	if user.Name == "" {
		return fmt.Errorf("name is empty")
	}

	if user.Email == "" {
		return fmt.Errorf("email is empty")
	}

	if user.Password == "" {
		return fmt.Errorf("password is empty")
	}

	if user.Salt == "" {
		return fmt.Errorf("salt is empty")
	}

	if user.Icon == "" {
		return fmt.Errorf("icon is empty")
	}

	user.ID = CreateUUID()
	user.Status = 1 // TODO: 状態確認

	if _, err := db.Insert(user); err != nil {
		return fmt.Errorf("Failed to create message object: %v", err)
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
