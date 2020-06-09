package bot

import (
	"bytes"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/service/bot/event"
	"go.uber.org/zap"
	"net/http"
	"sync"
	"time"
)

const (
	headerTRAQBotEvent             = "X-TRAQ-BOT-EVENT"
	headerTRAQBotRequestID         = "X-TRAQ-BOT-REQUEST-ID"
	headerTRAQBotVerificationToken = "X-TRAQ-BOT-TOKEN"
	headerUserAgent                = "User-Agent"
	ua                             = "traQ_Bot_Processor/1.0"
)

var eventSendCounter = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: "traq",
	Name:      "bot_event_send_count_total",
}, []string{"bot_id", "status"})

type dispatcherImpl struct {
	client http.Client
	l      *zap.Logger
	repo   repository.BotRepository
	wg     sync.WaitGroup
}

func initDispatcher(logger *zap.Logger, repo repository.BotRepository) Dispatcher {
	return &dispatcherImpl{
		client: http.Client{
			Jar:     nil,
			Timeout: 5 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		l:    logger.Named("bot.dispatcher"),
		repo: repo,
	}
}

func (d *dispatcherImpl) Send(b *model.Bot, event event.Type, body []byte) (ok bool) {
	d.wg.Add(1)
	defer d.wg.Done()

	reqID := uuid.Must(uuid.NewV4())

	req, _ := http.NewRequest(http.MethodPost, b.PostURL, bytes.NewReader(body))
	req.Header.Set(headerUserAgent, ua)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
	req.Header.Set(headerTRAQBotEvent, event.String())
	req.Header.Set(headerTRAQBotRequestID, reqID.String())
	req.Header.Set(headerTRAQBotVerificationToken, b.VerificationToken)

	start := time.Now()
	res, err := d.client.Do(req)
	stop := time.Now()

	if err != nil {
		eventSendCounter.WithLabelValues(b.ID.String(), "ne").Inc()
		d.writeLog(&model.BotEventLog{
			RequestID: reqID,
			BotID:     b.ID,
			Event:     event,
			Body:      string(body),
			Error:     err.Error(),
			Code:      -1,
			Latency:   stop.Sub(start).Nanoseconds(),
			DateTime:  time.Now(),
		})
		return false
	}
	_ = res.Body.Close()

	if res.StatusCode == http.StatusNoContent {
		eventSendCounter.WithLabelValues(b.ID.String(), "ok").Inc()
	} else {
		eventSendCounter.WithLabelValues(b.ID.String(), "ng").Inc()
	}

	d.writeLog(&model.BotEventLog{
		RequestID: reqID,
		BotID:     b.ID,
		Event:     event,
		Body:      string(body),
		Code:      res.StatusCode,
		Latency:   stop.Sub(start).Nanoseconds(),
		DateTime:  time.Now(),
	})
	return res.StatusCode == http.StatusNoContent
}

func (d *dispatcherImpl) Wait() {
	d.wg.Wait()
}

func (d *dispatcherImpl) writeLog(log *model.BotEventLog) {
	if err := d.repo.WriteBotEventLog(log); err != nil {
		d.l.Warn("failed to write log", zap.Error(err), zap.Any("eventLog", log))
	}
}
