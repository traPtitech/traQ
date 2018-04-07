package bot

import "github.com/satori/go.uuid"

// Store botデータ用ストア
type Store interface {
	WebhookStore
}

// WebhookStore Webhookデータ用ストア
type WebhookStore interface {
	SaveWebhook(webhook *Webhook) error
	GetAllWebhooks() []Webhook
	GetWebhook(id uuid.UUID) (Webhook, bool)
}
