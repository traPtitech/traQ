package model

import (
	"errors"
	"time"
)

const (
	// BotTypeGeneral : 汎用タイプBot
	BotTypeGeneral = 1
	// BotTypeWebhook : WebhookタイプBot
	BotTypeWebhook = 2
)

var (
	// ErrBotInvalidName : Botエラー botの名前が不正です。
	ErrBotInvalidName = errors.New("name must be 1-32 characters")

	// ErrBotRequireDescription : Botエラー botの説明は必須です。
	ErrBotRequireDescription = errors.New("description is required")
)

// Bot : Botの詳細構造体
type Bot struct {
	UserID      string    `xorm:"char(36) not null pk"`
	Type        int       `xorm:"int not null"`
	DisplayName string    `xorm:"varchar(32) not null"`
	Description string    `xorm:"text not null"`
	IsValid     bool      `xorm:"bool not null"`
	CreatorID   string    `xorm:"char(36) not null"`
	CreatedAt   time.Time `xorm:"created"`
	UpdaterID   string    `xorm:"char(36) not null"`
	UpdatedAt   time.Time `xorm:"updated"`
}

// TableName : Botのテーブル名
func (*Bot) TableName() string {
	return "bots"
}

// Update : Botを更新します
func (b *Bot) Update() (err error) {
	if len(b.DisplayName) == 0 || len(b.DisplayName) > 32 {
		return ErrBotInvalidName
	}

	_, err = db.ID(b.UserID).UseBool().Update(b)
	return
}

// Invalidate : Botを無効化します
func (b *Bot) Invalidate() (err error) {
	b.IsValid = false

	_, err = db.ID(b.UserID).UseBool().Update(b)
	return
}
