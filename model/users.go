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
	return "Users"
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

func (user *User) SetPassword(pass string) error {
	b := make([]byte, 14)

	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return fmt.Errorf("an error occurred while generating salt: %v", err)
	}

	user.Salt = hex.EncodeToString(b)
	user.Password = hashPassword(pass, user.Salt)

	return nil
}

func (user *User) Authorization(pass string) (bool, error) {
	if user.Name == "" {
		return false, fmt.Errorf("name is empty")
	}

	has, err := db.Where("name = ?", user.Name).Get(user)
	if err != nil {
		return false, fmt.Errorf("Failed to find message: %v", err)
	}
	if !has {
		user.Salt = "popopopopopo"
	}

	hashedPassword := hashPassword(pass, user.Salt)

	if subtle.ConstantTimeCompare([]byte(hashedPassword), []byte(user.Password)) != 1 {
		return false, fmt.Errorf("password or id is wrong")
	}
	return true, nil
}

func hashPassword(pass, salt string) string {
	converted := pbkdf2.Key([]byte(pass), []byte(salt), 65536, 64, sha512.New)
	return hex.EncodeToString(converted[:])
}
