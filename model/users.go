package model

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"

	"golang.org/x/crypto/scrypt"
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

	user.ID = createUUID()
	user.Status = 1 // TODO: 状態確認

	if _, err := db.Insert(user); err != nil {
		return fmt.Errorf("Failed to create message object: %v", err)
	}
	return nil
}

func (user *User) SetPassword(pass string) error {
	b := make([]byte, 14)

	_, err := io.ReadFull(rand.Reader, b)
	if err != nil {
		return fmt.Errorf("an error occurred while generating salt: %v", err)
	}

	user.Salt = base64.StdEncoding.EncodeToString(b)

	converted, err := scrypt.Key([]byte(pass), []byte(salt), 16384, 8, 1, 64)
	if err != nil {
		return fmt.Errorf("an error occurred while generating hashed password: %v", err)
	}

	user.Password = hex.EncodeToString(converted[:])
	return nil
}

func (user *User) Authorization(pass string) (bool, error) {
	if user.Name == "" {
		return false, fmt.Errorf("name is empty")
	}

	has, err := db.Get(user)
	if err != nil {
		return false, fmt.Errorf("Failed to find message: %v", err)
	}
	if !has {
		return false, fmt.Errorf("user is not found")
	}

	converted, err := scrypt.Key([]byte(pass), []byte(user.Salt), 16384, 8, 1, 64)
	if err != nil {
		return false, fmt.Errorf("an error occurred while checking hashed password: %v", err)
	}

	hashedPassword := hex.EncodeToString(converted[:])

	if hashedPassword != user.Password {
		return false, fmt.Errorf("password is wrong")
	}
	return true, nil
}
