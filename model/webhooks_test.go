package model

import (
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
)

func TestWebhookBot_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "webhook_bots", (&WebhookBot{}).TableName())
}

func TestWebhookBot_GetID(t *testing.T) {
	t.Parallel()

	t.Run("UUIDv4", func(tt *testing.T) {
		tt.Parallel()
		id := uuid.Must(uuid.NewV4())
		assert.Equal(tt, id, (&WebhookBot{ID: id}).GetID())
	})

	t.Run("UUIDv7", func(tt *testing.T) {
		tt.Parallel()
		id := uuid.Must(uuid.NewV7())
		assert.Equal(tt, id, (&WebhookBot{ID: id}).GetID())
	})
}

func TestWebhookBot_GetChannelID(t *testing.T) {

	t.Run("UUIDv4", func(tt *testing.T) {
		tt.Parallel()
		id := uuid.Must(uuid.NewV4())
		assert.Equal(tt, id, (&WebhookBot{ChannelID: id}).GetChannelID())
	})

	t.Run("UUIDv7", func(tt *testing.T) {
		tt.Parallel()
		id := uuid.Must(uuid.NewV7())
		assert.Equal(tt, id, (&WebhookBot{ChannelID: id}).GetChannelID())
	})
}

func TestWebhookBot_GetBotUserID(t *testing.T) {
	t.Parallel()
	t.Run("UUIDv4", func(tt *testing.T) {
		tt.Parallel()
		id := uuid.Must(uuid.NewV4())
		assert.Equal(tt, id, (&WebhookBot{BotUserID: id}).GetBotUserID())
	})

	t.Run("UUIDv7", func(tt *testing.T) {
		tt.Parallel()
		id := uuid.Must(uuid.NewV7())
		assert.Equal(tt, id, (&WebhookBot{BotUserID: id}).GetBotUserID())
	})
}

func TestWebhookBot_GetCreatorID(t *testing.T) {
	t.Parallel()
	t.Run("UUIDv4", func(tt *testing.T) {
		tt.Parallel()
		id := uuid.Must(uuid.NewV4())
		assert.Equal(tt, id, (&WebhookBot{CreatorID: id}).GetCreatorID())
	})

	t.Run("UUIDv7", func(tt *testing.T) {
		tt.Parallel()
		id := uuid.Must(uuid.NewV7())
		assert.Equal(tt, id, (&WebhookBot{CreatorID: id}).GetCreatorID())
	})
}

func TestWebhookBot_GetDescription(t *testing.T) {
	t.Parallel()
	t.Run("UUIDv4", func(tt *testing.T) {
		tt.Parallel()
		desc := "test"
		assert.Equal(tt, desc, (&WebhookBot{Description: desc}).GetDescription())
	})

	t.Run("UUIDv7", func(tt *testing.T) {
		tt.Parallel()
		desc := "test"
		assert.Equal(tt, desc, (&WebhookBot{Description: desc}).GetDescription())
	})
}

func TestWebhookBot_GetSecret(t *testing.T) {
	t.Parallel()
	secret := "secret"
	assert.Equal(t, secret, (&WebhookBot{Secret: secret}).GetSecret())
}

func TestWebhookBot_GetName(t *testing.T) {
	t.Parallel()
	name := "test"
	assert.Equal(t, name, (&WebhookBot{BotUser: User{DisplayName: name}}).GetName())
}

func TestWebhookBot_GetCreatedAt(t *testing.T) {
	t.Parallel()
	tm := time.Now()
	assert.Equal(t, tm, (&WebhookBot{CreatedAt: tm}).GetCreatedAt())
}

func TestWebhookBot_GetUpdatedAt(t *testing.T) {
	t.Parallel()
	tm := time.Now()
	assert.Equal(t, tm, (&WebhookBot{UpdatedAt: tm}).GetUpdatedAt())
}
