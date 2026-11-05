package event

import (
	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/model"
)

// WebhookCreatedPayload WebhookCreated のペイロード
type WebhookCreatedPayload struct {
	WebhookID uuid.UUID         `hub:"webhook_id"`
	Webhook   *model.WebhookBot `hub:"webhook"`
}

func (WebhookCreatedPayload) TopicName() string { return WebhookCreated }

// WebhookUpdatedPayload WebhookUpdated のペイロード
type WebhookUpdatedPayload struct {
	WebhookID uuid.UUID `hub:"webhook_id"`
}

func (WebhookUpdatedPayload) TopicName() string { return WebhookUpdated }

// WebhookDeletedPayload WebhookDeleted のペイロード
type WebhookDeletedPayload struct {
	WebhookID uuid.UUID `hub:"webhook_id"`
}

func (WebhookDeletedPayload) TopicName() string { return WebhookDeleted }

var (
	_ Payload = WebhookCreatedPayload{}
	_ Payload = WebhookUpdatedPayload{}
	_ Payload = WebhookDeletedPayload{}
)
