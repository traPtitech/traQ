//go:generate mockgen -source=$GOFILE -destination=mock_$GOPACKAGE/mock_$GOFILE
package handler

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/service/bot/event"
	"github.com/traPtitech/traQ/service/channel"
	"go.uber.org/zap"
)

type Context interface {
	CM() channel.Manager
	R() repository.Repository
	L() *zap.Logger
	D() event.Dispatcher

	GetBot(id uuid.UUID) (*model.Bot, error)
	GetBotByBotUserID(uid uuid.UUID) (*model.Bot, error)
	GetBots(event model.BotEventType) ([]*model.Bot, error)
	GetChannelBots(cid uuid.UUID, event model.BotEventType) ([]*model.Bot, error)
}
