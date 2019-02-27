package model

import (
	"github.com/satori/go.uuid"
	"time"
)

// Webhook Webhook
type Webhook interface {
	GetID() uuid.UUID
	GetBotUserID() uuid.UUID
	GetName() string
	GetDescription() string
	GetChannelID() uuid.UUID
	GetCreatorID() uuid.UUID
	GetCreatedAt() time.Time
	GetUpdatedAt() time.Time
}

// WebhookBot DB用WebhookBot構造体
type WebhookBot struct {
	ID          uuid.UUID  `gorm:"type:char(36);not null;primary_key"`
	BotUserID   uuid.UUID  `gorm:"type:char(36);not null;unique"`
	BotUser     User       `gorm:"foreignkey:BotUserID"`
	Description string     `gorm:"type:text;not null"`
	ChannelID   uuid.UUID  `gorm:"type:char(36);not null"`
	CreatorID   uuid.UUID  `gorm:"type:char(36);not null"`
	CreatedAt   time.Time  `gorm:"precision:6"`
	UpdatedAt   time.Time  `gorm:"precision:6"`
	DeletedAt   *time.Time `gorm:"precision:6"`
}

// TableName Webhookのテーブル名
func (*WebhookBot) TableName() string {
	return "webhook_bots"
}

// GetID WebhookIDを返します
func (w *WebhookBot) GetID() uuid.UUID {
	return w.ID
}

// GetBotUserID WebhookUserのIDを返します
func (w *WebhookBot) GetBotUserID() uuid.UUID {
	return w.BotUserID
}

// GetName Webhookの名前を返します
func (w *WebhookBot) GetName() string {
	return w.BotUser.Name
}

// GetDescription Webhookの説明を返します
func (w *WebhookBot) GetDescription() string {
	return w.Description
}

// GetChannelID Webhookのデフォルト投稿チャンネルのIDを返します
func (w *WebhookBot) GetChannelID() uuid.UUID {
	return w.ChannelID
}

// GetCreatorID Webhookの製作者IDを返します
func (w *WebhookBot) GetCreatorID() uuid.UUID {
	return w.CreatorID
}

// GetCreatedAt Webhookの作成日時を返します
func (w *WebhookBot) GetCreatedAt() time.Time {
	return w.CreatedAt
}

// GetUpdatedAt Webhookの更新日時を返します
func (w *WebhookBot) GetUpdatedAt() time.Time {
	return w.UpdatedAt
}
