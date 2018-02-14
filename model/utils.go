package model

import (
	"fmt"

	"github.com/go-xorm/xorm"
	"github.com/satori/go.uuid"
)

var (
	db *xorm.Engine
)

// SetXORMEngine DBにxormのエンジンを設定する
func SetXORMEngine(engine *xorm.Engine) {
	db = engine
}

// SyncSchema : テーブルと構造体を同期させる関数
// モデルを追加したら各自ここに追加しなければいけない
func SyncSchema() error {
	if err := db.Sync(&Channel{}); err != nil {
		return fmt.Errorf("Failed to sync Channels Table: %v", err)
	}

	if err := db.Sync(&UsersPrivateChannel{}); err != nil {
		return fmt.Errorf("Failed to sync UsersPrivateChannels Table: %v", err)
	}

	if err := db.Sync(&Message{}); err != nil {
		return fmt.Errorf("Failed to sync Messages Table: %v", err)
	}

	if err := db.Sync(&User{}); err != nil {
		return fmt.Errorf("Failed to sync Users Table: %v", err)
	}

	if err := db.Sync(&Clip{}); err != nil {
		return fmt.Errorf("Failed to sync Clips Table: %v", err)
	}
	if err := db.Sync(&UsersTag{}); err != nil {
		return fmt.Errorf("Failed to sync users_tags Table: %v", err)
	}
	if err := db.Sync(&Tag{}); err != nil {
		return fmt.Errorf("Failed to sync tags Table: %v", err)
	}

	if err := db.Sync(&Star{}); err != nil {
		return fmt.Errorf("Failed to sync Stars Table: %v", err)
	}

	if err := db.Sync(&Pin{}); err != nil {
		return fmt.Errorf("Failed to sync Pins Table: %v", err)
	}

	traq := &User{
		Name:  "traq",
		Email: "trap.titech@gmail.com",
		Icon:  "Empty",
	}
	ok, err := traq.Exists()
	if err != nil {
		return err
	}
	if !ok {
		traq.SetPassword("traq")
		traq.Create()
	}

	return nil
}

// CreateUUID UUIDを生成する
func CreateUUID() string {
	return uuid.NewV4().String()
}
