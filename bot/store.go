package bot

import "github.com/satori/go.uuid"

// Store botデータ用ストア
type Store interface {
	WebhookStore
	GeneralBotStore
	PluginStore
}

// WebhookStore Webhookデータ用ストア
type WebhookStore interface {
	SaveWebhook(webhook *Webhook) error
	GetAllWebhooks() []Webhook
	GetWebhook(id uuid.UUID) (Webhook, bool)
}

// GeneralBotStore GeneralBotデータ用ストア
type GeneralBotStore interface {
	SaveGeneralBot(bot *GeneralBot) error
	GetAllGeneralBots() []GeneralBot
	GetGeneralBot(id uuid.UUID) (GeneralBot, bool)
	GetInstalledChannels(botID uuid.UUID) []InstalledChannel
	GetInstalledBot(channelID uuid.UUID) []InstalledChannel
	InstallBot(botID, channelID, userID uuid.UUID) error
	UninstallBot(botID, channelID uuid.UUID) error
}

type PluginStore interface {
}
