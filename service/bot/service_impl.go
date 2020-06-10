package bot

import (
	"context"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/service/bot/event"
	"github.com/traPtitech/traQ/service/channel"
	"go.uber.org/zap"
	"sync"
)

type serviceImpl struct {
	repo       repository.Repository
	cm         channel.Manager
	logger     *zap.Logger
	dispatcher event.Dispatcher
	hub        *hub.Hub

	sub     hub.Subscription
	wg      sync.WaitGroup
	started bool
}

// NewService ボットサービスを生成します
func NewService(repo repository.Repository, cm channel.Manager, hub *hub.Hub, logger *zap.Logger) Service {
	p := &serviceImpl{
		repo:       repo,
		cm:         cm,
		logger:     logger.Named("bot"),
		hub:        hub,
		dispatcher: event.NewDispatcher(logger, repo),
	}
	return p
}

func (p *serviceImpl) Start() {
	if p.started {
		return
	}
	p.started = true

	events := make([]string, 0, len(eventHandlerSet))
	for k := range eventHandlerSet {
		events = append(events, k)
	}
	p.sub = p.hub.Subscribe(100, events...)

	go func() {
		for ev := range p.sub.Receiver {
			p.wg.Add(1)
			go func(ev hub.Message) {
				defer p.wg.Done()
				h, ok := eventHandlerSet[ev.Name]
				if ok {
					h(p, ev.Name, ev.Fields)
				}
			}(ev)
		}
	}()
	p.logger.Info("bot service started")
}

func (p *serviceImpl) Shutdown(ctx context.Context) error {
	if !p.started {
		return nil
	}
	p.hub.Unsubscribe(p.sub)
	p.wg.Wait()
	p.logger.Info("bot service shutdown")
	return nil
}

func (p *serviceImpl) CM() channel.Manager {
	return p.cm
}

func (p *serviceImpl) R() repository.Repository {
	return p.repo
}

func (p *serviceImpl) L() *zap.Logger {
	return p.logger
}

func (p *serviceImpl) D() event.Dispatcher {
	return p.dispatcher
}
