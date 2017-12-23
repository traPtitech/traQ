package model

import (
	"fmt"

	"github.com/go-xorm/xorm"
	"github.com/satori/go.uuid"
)

var db *xorm.Engine

// SetXORMEngine DBにxormのエンジンを設定する
func SetXORMEngine(engine *xorm.Engine) {
	db = engine
}

// SyncSchema : テーブルと構造体を同期させる関数
// モデルを追加したら各自ここに追加しなければいけない
func SyncSchema() error {
	if err := db.Sync(new(Channel)); err != nil {
		return fmt.Errorf("Failed to sync Channels: %v", err)
	}

	if err := db.Sync(new(UsersPrivateChannel)); err != nil {
		return fmt.Errorf("Failed to sync UsersPrivateChannels: %v", err)
	}

	if err := db.Sync(&Message{}); err != nil {
		return fmt.Errorf("Failed to sync Messages Table: %v", err)
	}
	return nil
}

// CreateUUID UUIDを生成する
func CreateUUID() string {
	return uuid.NewV4().String()
}
