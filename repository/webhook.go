package repository

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
	"gopkg.in/guregu/null.v3"
)

// UpdateWebhookArgs Webhook情報更新引数
type UpdateWebhookArgs struct {
	Name        null.String
	Description null.String
	ChannelID   uuid.NullUUID
	Secret      null.String
}

// WebhookRepository Webhookボットリポジトリ
type WebhookRepository interface {
	CreateWebhook(name, description string, channelID, creatorID uuid.UUID, secret string) (model.Webhook, error)
	UpdateWebhook(id uuid.UUID, args UpdateWebhookArgs) error
	DeleteWebhook(id uuid.UUID) error
	GetWebhook(id uuid.UUID) (model.Webhook, error)
	GetWebhookByBotUserID(id uuid.UUID) (model.Webhook, error)
	GetAllWebhooks() ([]model.Webhook, error)
	GetWebhooksByCreator(creatorID uuid.UUID) ([]model.Webhook, error)
}
