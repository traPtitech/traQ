package repository

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
	"gopkg.in/guregu/null.v3"
)

// UpdateBotArgs Bot情報更新引数
type UpdateBotArgs struct {
	DisplayName null.String
	Description null.String
	WebhookURL  null.String
	Privileged  null.Bool
}

// BotRepository Botリポジトリ
type BotRepository interface {
	CreateBot(name, displayName, description string, creatorID uuid.UUID, webhookURL string) (*model.Bot, error)
	UpdateBot(id uuid.UUID, args UpdateBotArgs) error
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
