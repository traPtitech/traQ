package repository

import (
	"database/sql"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/model"
)

// UpdateWebhookArgs Webhook情報更新引数
type UpdateWebhookArgs struct {
	Name        sql.NullString
	Description sql.NullString
	ChannelID   uuid.NullUUID
	Secret      sql.NullString
}

// WebhookRepository Webhookボットリポジトリ
type WebhookRepository interface {
	CreateWebhook(name, description string, channelID, creatorID uuid.UUID, secret string) (model.Webhook, error)
	UpdateWebhook(id uuid.UUID, args UpdateWebhookArgs) error
	DeleteWebhook(id uuid.UUID) error
	GetWebhook(id uuid.UUID) (model.Webhook, error)
	GetAllWebhooks() ([]model.Webhook, error)
	GetWebhooksByCreator(creatorID uuid.UUID) ([]model.Webhook, error)
}
