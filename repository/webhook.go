package repository

import (
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/model"
)

// WebhookRepository Webhookボットリポジトリ
type WebhookRepository interface {
	CreateWebhook(name, description string, channelID, creatorID, iconFileID uuid.UUID) (model.Webhook, error)
	UpdateWebhook(id uuid.UUID, name, description *string, channelID uuid.UUID) error
	DeleteWebhook(id uuid.UUID) error
	GetWebhook(id uuid.UUID) (model.Webhook, error)
	GetAllWebhooks() ([]model.Webhook, error)
	GetWebhooksByCreator(creatorID uuid.UUID) ([]model.Webhook, error)
}
