package bot

import (
	"bytes"
	"encoding/json"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"go.uber.org/zap"
	"net/http"
	"sync"
	"time"
)

const (
	headerTRAQBotEvent             = "X-TRAQ-BOT-EVENT"
	headerTRAQBotRequestID         = "X-TRAQ-BOT-REQUEST-ID"
	headerTRAQBotVerificationToken = "X-TRAQ-BOT-TOKEN"
)

// Processor ボットプロセッサー
type Processor struct {
	repo    repository.Repository
	logger  *zap.Logger
	hub     *hub.Hub
	bufPool sync.Pool
	client  http.Client
}

// NewProcessor ボットプロセッサーを生成し、起動します
func NewProcessor(repo repository.Repository, hub *hub.Hub, logger *zap.Logger) *Processor {
	p := &Processor{
		repo:   repo,
		logger: logger,
		hub:    hub,
		bufPool: sync.Pool{
			New: func() interface{} { return &bytes.Buffer{} },
		},
		client: http.Client{
			Timeout:       5 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse },
		},
	}
	go func() {
		events := make([]string, 0, len(eventHandlers))
		for k := range eventHandlers {
			events = append(events, k)
		}

		sub := hub.Subscribe(100, events...)
		for ev := range sub.Receiver {
			h, ok := eventHandlers[ev.Name]
			if ok {
				go h(p, ev.Name, ev.Fields)
			}
		}
	}()
	return p
}

func (p *Processor) sendEvent(b *model.Bot, event model.BotEvent, body []byte) (ok bool) {
	reqID := uuid.Must(uuid.NewV4())

	req, _ := http.NewRequest(http.MethodPost, b.PostURL, bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
	req.Header.Set(headerTRAQBotEvent, event.String())
	req.Header.Set(headerTRAQBotRequestID, reqID.String())
	req.Header.Set(headerTRAQBotVerificationToken, b.VerificationToken)

	res, err := p.client.Do(req)
	if err != nil {
		p.logger.Error("failed to send bot event. network error", zap.Error(err))
		if err := p.repo.WriteBotEventLog(&model.BotEventLog{
			RequestID: reqID,
			BotID:     b.ID,
			Event:     event,
			Code:      -1,
			DateTime:  time.Now(),
		}); err != nil {
			p.logger.Error("failed to WriteBotEventLog", zap.Error(err), zap.Stringer("requestId", reqID))
		}
		return false
	}
	_ = res.Body.Close()

	if err := p.repo.WriteBotEventLog(&model.BotEventLog{
		RequestID: reqID,
		BotID:     b.ID,
		Event:     event,
		Code:      res.StatusCode,
		DateTime:  time.Now(),
	}); err != nil {
		p.logger.Error("failed to WriteBotEventLog", zap.Error(err), zap.Stringer("requestId", reqID))
	}

	return res.StatusCode == http.StatusNoContent
}

func (p *Processor) makePayloadJSON(payload interface{}) (b []byte, releaseFunc func(), err error) {
	buf := p.bufPool.Get().(*bytes.Buffer)
	releaseFunc = func() {
		buf.Reset()
		p.bufPool.Put(buf)
	}

	if err := json.NewEncoder(buf).Encode(&payload); err != nil {
		releaseFunc()
		return nil, nil, err
	}

	return buf.Bytes(), releaseFunc, nil
}
