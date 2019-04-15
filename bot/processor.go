package bot

import (
	"bytes"
	"encoding/json"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/utils/message"
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
		sub := hub.Subscribe(10, event.MessageCreated)
		for ev := range sub.Receiver {
			m := ev.Fields["message"].(*model.Message)
			e := ev.Fields["embedded"].([]*message.EmbeddedInfo)
			plain := ev.Fields["plain"].(string)
			go p.createMessageHandler(m, e, plain)
		}
	}()
	go func() {
		sub := hub.Subscribe(1, event.BotPingRequest)
		for ev := range sub.Receiver {
			botID := ev.Fields["bot_id"].(uuid.UUID)
			bot, err := repo.GetBotByID(botID)
			if err != nil {
				logger.Error("failed to GetBotByID", zap.Error(err), zap.Stringer("bot_id", botID))
				continue
			}
			p.pingHandler(bot)
		}
	}()
	go func() {
		sub := hub.Subscribe(10, event.BotJoined, event.BotLeft)
		for ev := range sub.Receiver {
			botID := ev.Fields["bot_id"].(uuid.UUID)
			chID := ev.Fields["channel_id"].(uuid.UUID)
			switch ev.Name {
			case event.BotJoined:
				go p.joinedAndLeftHandler(botID, chID, Joined)
			case event.BotLeft:
				go p.joinedAndLeftHandler(botID, chID, Left)
			}
		}
	}()
	go func() {
		sub := hub.Subscribe(100,
			event.ChannelCreated,
			event.UserCreated,
		)
		for ev := range sub.Receiver {
			switch ev.Name {
			case event.ChannelCreated:
				go p.channelCreatedHandler(ev.Fields["channel_id"].(uuid.UUID), ev.Fields["private"].(bool))
			case event.UserCreated:
				go p.userCreatedHandler(ev.Fields["user"].(*model.User))
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
