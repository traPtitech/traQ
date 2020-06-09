package bot

import (
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/service/channel"
	"go.uber.org/zap"
)

// Processor ボットプロセッサー
type Processor struct {
	repo       repository.Repository
	cm         channel.Manager
	logger     *zap.Logger
	hub        *hub.Hub
	dispatcher Dispatcher
}

// NewProcessor ボットプロセッサーを生成し、起動します
func NewProcessor(repo repository.Repository, cm channel.Manager, hub *hub.Hub, logger *zap.Logger) *Processor {
	p := &Processor{
		repo:       repo,
		cm:         cm,
		logger:     logger.Named("bot"),
		hub:        hub,
		dispatcher: initDispatcher(logger, repo),
	}
	go func() {
		events := make([]string, 0, len(eventHandlerSet))
		for k := range eventHandlerSet {
			events = append(events, k)
		}

		sub := hub.Subscribe(100, events...)
		for ev := range sub.Receiver {
			h, ok := eventHandlerSet[ev.Name]
			if ok {
				go h(p, ev.Name, ev.Fields)
			}
		}
	}()
	return p
}
