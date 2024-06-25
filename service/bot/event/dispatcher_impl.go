package event

import (
	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	botWS "github.com/traPtitech/traQ/service/bot/ws"
)

var eventSendCounter = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: "traq",
	Name:      "bot_event_send_count_total",
}, []string{"bot_id", "status"})

const (
	resultOK           = "ok"
	resultNG           = "ng"
	resultNetworkError = "ne"
	resultDropped      = "dp"
)

type dispatcherImpl struct {
	http *httpDispatcher
	ws   *wsDispatcher
	l    *zap.Logger
	repo repository.BotRepository
}

func NewDispatcher(logger *zap.Logger, repo repository.BotRepository, s *botWS.Streamer) Dispatcher {
	return &dispatcherImpl{
		http: newHTTPDispatcher(logger),
		ws:   newWSDispatcher(s, logger),
		l:    logger.Named("bot.dispatcher"),
		repo: repo,
	}
}

func (d *dispatcherImpl) Send(b *model.Bot, event model.BotEventType, body []byte) (ok bool) {
	reqID := uuid.Must(uuid.NewV7())

	var log *model.BotEventLog
	switch b.Mode {
	case model.BotModeHTTP:
		ok, log = d.http.send(b, event, reqID, body)
	case model.BotModeWebSocket:
		ok, log = d.ws.send(b, event, reqID, body)
	default:
		return false
	}

	d.writeLog(log)
	return ok
}

func (d *dispatcherImpl) writeLog(log *model.BotEventLog) {
	if err := d.repo.WriteBotEventLog(log); err != nil {
		d.l.Warn("failed to write log", zap.Error(err), zap.Any("eventLog", log))
	}
}
