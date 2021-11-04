package event

import (
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/zap"

	"github.com/traPtitech/traQ/model"
	botWS "github.com/traPtitech/traQ/service/bot/ws"
)

type wsDispatcher struct {
	s *botWS.Streamer
	l *zap.Logger
}

func newWSDispatcher(s *botWS.Streamer, logger *zap.Logger) *wsDispatcher {
	return &wsDispatcher{
		s: s,
		l: logger.Named("bot.dispatcher.ws"),
	}
}

func (d *wsDispatcher) send(b *model.Bot, event model.BotEventType, reqID uuid.UUID, body []byte) (ok bool, log *model.BotEventLog) {
	start := time.Now()
	errs, attempted := d.s.WriteMessage(event.String(), reqID, body, b.BotUserID)
	latency := time.Since(start)

	log = &model.BotEventLog{
		RequestID: reqID,
		BotID:     b.ID,
		Event:     event,
		Body:      string(body),
		Latency:   latency.Nanoseconds(),
		DateTime:  start,
	}
	if len(errs) > 0 {
		eventSendCounter.WithLabelValues(b.ID.String(), resultNetworkError).Inc()
		log.Result = resultNetworkError
		log.Error = formatErrors(errs)
		return false, log
	}
	if !attempted {
		eventSendCounter.WithLabelValues(b.ID.String(), resultOK).Inc()
		log.Result = resultDropped
		return false, log
	}
	eventSendCounter.WithLabelValues(b.ID.String(), resultOK).Inc()
	log.Result = resultOK
	return true, log
}

func formatErrors(errs []error) string {
	var b strings.Builder
	for _, err := range errs {
		b.WriteString(err.Error())
	}
	return b.String()
}
