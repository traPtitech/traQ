package repository

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
)

// BotRepository Botリポジトリ
type BotRepository interface {
	CreateBot(name, displayName, description string, creatorID uuid.UUID, webhookURL string) (*model.Bot, error)
	SetSubscribeEventsToBot(botID uuid.UUID, events model.BotEvents) error
	GetAllBots() ([]*model.Bot, error)
	GetBotByID(id uuid.UUID) (*model.Bot, error)
	GetBotByCode(code string) (*model.Bot, error)
	GetBotsByCreator(userID uuid.UUID) ([]*model.Bot, error)
	GetBotsByChannel(channelID uuid.UUID) ([]*model.Bot, error)
	ChangeBotState(id uuid.UUID, state model.BotState) error
	ReissueBotTokens(id uuid.UUID) (*model.Bot, error)
	DeleteBot(id uuid.UUID) error
	AddBotToChannel(botID, channelID uuid.UUID) error
	RemoveBotFromChannel(botID, channelID uuid.UUID) error
	GetParticipatingChannelIDsByBot(botID uuid.UUID) ([]uuid.UUID, error)
	WriteBotEventLog(log *model.BotEventLog) error
	GetBotEventLogs(botID uuid.UUID, limit, offset int) ([]*model.BotEventLog, error)
}
