package handler

import (
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
}
