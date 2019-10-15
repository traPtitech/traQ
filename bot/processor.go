package bot

import (
	"bytes"
	"github.com/gofrs/uuid"
	jsoniter "github.com/json-iterator/go"
	"github.com/labstack/echo"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"go.uber.org/zap"
	"net/http"
	"time"
)

const (
	headerTRAQBotEvent             = "X-TRAQ-BOT-EVENT"
	headerTRAQBotRequestID         = "X-TRAQ-BOT-REQUEST-ID"
	headerTRAQBotVerificationToken = "X-TRAQ-BOT-TOKEN"
)

// Processor ボットプロセッサー
type Processor struct {
	repo   repository.Repository
	logger *zap.Logger
	hub    *hub.Hub
	client http.Client
}

// NewProcessor ボットプロセッサーを生成し、起動します
func NewProcessor(repo repository.Repository, hub *hub.Hub, logger *zap.Logger) *Processor {
	p := &Processor{
		repo:   repo,
		logger: logger,
		hub:    hub,
		client: http.Client{
			Timeout:       10 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse },
		},
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

func (p *Processor) sendEvent(b *model.Bot, event model.BotEvent, body []byte) (ok bool) {
	reqID := uuid.Must(uuid.NewV4())

	req, _ := http.NewRequest(http.MethodPost, b.PostURL, bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
	req.Header.Set(headerTRAQBotEvent, event.String())
	req.Header.Set(headerTRAQBotRequestID, reqID.String())
	req.Header.Set(headerTRAQBotVerificationToken, b.VerificationToken)

	start := time.Now()
	res, err := p.client.Do(req)
	stop := time.Now()

	if err != nil {
		p.logger.Error("failed to send bot event. network error", zap.Error(err))
		if err := p.repo.WriteBotEventLog(&model.BotEventLog{
			RequestID: reqID,
			BotID:     b.ID,
			Event:     event,
			Body:      string(body),
			Error:     err.Error(),
			Code:      -1,
			Latency:   stop.Sub(start).Nanoseconds(),
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
		Body:      string(body),
		Code:      res.StatusCode,
		Latency:   stop.Sub(start).Nanoseconds(),
		DateTime:  time.Now(),
	}); err != nil {
		p.logger.Error("failed to WriteBotEventLog", zap.Error(err), zap.Stringer("requestId", reqID))
	}

	return res.StatusCode == http.StatusNoContent
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
