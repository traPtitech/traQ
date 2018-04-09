package bot

import "github.com/satori/go.uuid"

// Store botデータ用ストア
type Store interface {
	WebhookStore
	GeneralBotStore
	PluginStore
	SavePostLog(reqID, botUserID uuid.UUID, status int, request, response, error string) error
}

// WebhookStore Webhookデータ用ストア
type WebhookStore interface {
	SaveWebhook(webhook *Webhook) error
	UpdateWebhook(webhook *Webhook) error
	GetAllWebhooks() ([]Webhook, error)
}

// GeneralBotStore GeneralBotデータ用ストア
type GeneralBotStore interface {
	SaveGeneralBot(bot *GeneralBot) error
	UpdateGeneralBot(bot *GeneralBot) error
	GetAllGeneralBots() ([]GeneralBot, error)
	GetAllBotsInstalledChannels() ([]InstalledChannel, error)
	InstallBot(botID, channelID, userID uuid.UUID) error
	UninstallBot(botID, channelID uuid.UUID) error
}

// PluginStore Pluginデータ用ストア
type PluginStore interface {
	SavePlugin(plugin *Plugin) error
	UpdatePlugin(plugin *Plugin) error
	GetAllPlugins() ([]Plugin, error)
}
