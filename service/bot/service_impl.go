package bot

import (
	"context"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"
	"github.com/lthibault/jitterbug/v2"
	"go.uber.org/zap"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/service/bot/event"
	botWS "github.com/traPtitech/traQ/service/bot/ws"
	"github.com/traPtitech/traQ/service/channel"
)

const (
	botEventLogPurgeBefore = time.Hour * 24 * 365 // BOTイベントログを1年間保持
)

type serviceImpl struct {
	repo       repository.Repository
	cm         channel.Manager
	logger     *zap.Logger
	dispatcher event.Dispatcher
	hub        *hub.Hub

	sub         hub.Subscription
	logPurger   *jitterbug.Ticker
	serviceDone chan struct{}
	hubDone     chan struct{}
	purgerDone  chan struct{}
}

// NewService ボットサービスを生成します
func NewService(repo repository.Repository, cm channel.Manager, hub *hub.Hub, s *botWS.Streamer, logger *zap.Logger) Service {
	p := &serviceImpl{
		repo:       repo,
		cm:         cm,
		logger:     logger.Named("bot"),
		hub:        hub,
		dispatcher: event.NewDispatcher(logger, repo, s),

		serviceDone: make(chan struct{}),
		hubDone:     make(chan struct{}),
		purgerDone:  make(chan struct{}),
	}
	p.start()
	return p
}

func (p *serviceImpl) start() {
	// イベントの発送を開始
	events := make([]string, 0, len(eventHandlerSet))
	for k := range eventHandlerSet {
		events = append(events, k)
	}
	p.sub = p.hub.Subscribe(100, events...)

	go func() {
		defer close(p.hubDone)
		var wg sync.WaitGroup
		for ev := range p.sub.Receiver {
			wg.Add(1)
			go func(ev hub.Message) {
				defer wg.Done()
				h, ok := eventHandlerSet[ev.Name]
				if ok {
					err := h(p, time.Now(), ev.Name, ev.Fields)
					if err != nil {
						p.logger.Error("an error occurred while processing event", zap.Error(err), zap.String("event", ev.Name))
					}
				}
			}(ev)
		}
		wg.Wait()
	}()

	// BOTイベントログの定期的消去
	p.logPurger = jitterbug.New(time.Hour*24, &jitterbug.Uniform{
		Min: time.Hour * 23,
	})
	go func() {
		defer close(p.purgerDone)
		for {
			select {
			case _, ok := <-p.logPurger.C:
				if !ok {
					return
				}
				if err := p.repo.PurgeBotEventLogs(time.Now().Add(-botEventLogPurgeBefore)); err != nil {
					p.logger.Error("an error occurred while puring old bot event logs", zap.Error(err))
				}
			case <-p.serviceDone:
				return
			}
		}
	}()

	p.logger.Info("bot service started")
}

func (p *serviceImpl) Shutdown(ctx context.Context) error {
	p.hub.Unsubscribe(p.sub)
	p.logPurger.Stop()
	close(p.serviceDone)
	<-p.hubDone
	<-p.purgerDone
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

func (p *serviceImpl) Unicast(ev model.BotEventType, payload interface{}, target *model.Bot) error {
	return event.Unicast(p.dispatcher, ev, payload, target)
}

func (p *serviceImpl) Multicast(ev model.BotEventType, payload interface{}, targets []*model.Bot) error {
	return event.Multicast(p.dispatcher, ev, payload, targets)
}

func (p *serviceImpl) GetBot(id uuid.UUID) (*model.Bot, error) {
	bots, err := p.repo.GetBots(repository.BotsQuery{}.Active().BotID(id))
	if err != nil {
		return nil, err
	}
	if len(bots) == 0 {
		return nil, nil
	}
	return bots[0], nil
}

func (p *serviceImpl) GetBotByBotUserID(uid uuid.UUID) (*model.Bot, error) {
	bots, err := p.repo.GetBots(repository.BotsQuery{}.Active().BotUserID(uid))
	if err != nil {
		return nil, err
	}
	if len(bots) == 0 {
		return nil, nil
	}
	return bots[0], nil
}

func (p *serviceImpl) GetBots(event model.BotEventType) ([]*model.Bot, error) {
	return p.repo.GetBots(repository.BotsQuery{}.Active().Subscribe(event))
}

func (p *serviceImpl) GetChannelBots(cid uuid.UUID, event model.BotEventType) ([]*model.Bot, error) {
	return p.repo.GetBots(repository.BotsQuery{}.Active().Subscribe(event).CMemberOf(cid))
}
