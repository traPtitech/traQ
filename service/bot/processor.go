package bot

import (
	"github.com/gofrs/uuid"
	jsoniter "github.com/json-iterator/go"
	"github.com/leandro-lugaresi/hub"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/service/bot/event"
	"github.com/traPtitech/traQ/service/channel"
	"go.uber.org/zap"
	"sync"
)

var eventSendCounter = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: "traq",
	Name:      "bot_event_send_count_total",
}, []string{"bot_id", "status"})

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

func (p *Processor) makePayloadJSON(payload interface{}) (b []byte, releaseFunc func(), err error) {
	cfg := jsoniter.ConfigFastest
	stream := cfg.BorrowStream(nil)
	releaseFunc = func() { cfg.ReturnStream(stream) }
	stream.WriteVal(payload)
	stream.WriteRaw("\n")
	if err = stream.Error; err != nil {
		releaseFunc()
		return nil, nil, err
	}
	return stream.Buffer(), releaseFunc, nil
}

func (p *Processor) unicast(ev event.Type, payload interface{}, target *model.Bot) {
	buf, release, err := p.makePayloadJSON(&payload)
	if err != nil {
		p.logger.Error("unexpected json encode error", zap.Error(err))
		return
	}
	defer release()

	p.dispatcher.Send(target, ev, buf)
}

func (p *Processor) multicast(ev event.Type, payload interface{}, targets []*model.Bot) {
	if len(targets) == 0 {
		return
	}
	buf, release, err := p.makePayloadJSON(&payload)
	if err != nil {
		p.logger.Error("unexpected json encode error", zap.Error(err))
		return
	}
	defer release()

	var wg sync.WaitGroup
	done := make(map[uuid.UUID]bool, len(targets))
	for _, bot := range targets {
		if !done[bot.ID] {
			done[bot.ID] = true
			bot := bot
			wg.Add(1)
			go func() {
				defer wg.Done()
				p.dispatcher.Send(bot, ev, buf)
			}()
		}
	}
	wg.Wait()
}
