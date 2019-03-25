package repository

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
)

// BotRepository Botリポジトリ
type BotRepository interface {
	CreateBot(name, displayName, description string, creatorID uuid.UUID, webhookURL string) (*model.Bot, error)
	SetSubscribeEventsToBot(botID uuid.UUID, events model.BotEvents) error
	GetBotByID(id uuid.UUID) (*model.Bot, error)
	GetBotByCode(code string) (*model.Bot, error)
	GetBotsByCreator(userID uuid.UUID) ([]*model.Bot, error)
	GetBotsByChannel(channelID uuid.UUID) ([]*model.Bot, error)
	ChangeBotStatus(id uuid.UUID, status model.BotStatus) error
	DeleteBot(id uuid.UUID) error
	AddBotToChannel(botID, channelID uuid.UUID) error
	RemoveBotFromChannel(botID, channelID uuid.UUID) error
	GetParticipatingChannelIDsByBot(botID uuid.UUID) ([]uuid.UUID, error)
}
