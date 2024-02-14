package event

import (
	"bytes"
	"net/http"
	"time"

	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"github.com/traPtitech/traQ/model"
)

const (
	headerTRAQBotEvent             = "X-TRAQ-BOT-EVENT"
	headerTRAQBotRequestID         = "X-TRAQ-BOT-REQUEST-ID"
	headerTRAQBotVerificationToken = "X-TRAQ-BOT-TOKEN"
	headerUserAgent                = "User-Agent"
	ua                             = "traQ_Bot_Processor/1.0"
)

type httpDispatcher struct {
	client http.Client
	l      *zap.Logger
}

func newHTTPDispatcher(logger *zap.Logger) *httpDispatcher {
	return &httpDispatcher{
		client: http.Client{
			Jar:     nil,
			Timeout: 5 * time.Second,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		l: logger.Named("bot.dispatcher.http"),
	}
}

func (d *httpDispatcher) send(b *model.Bot, event model.BotEventType, reqID uuid.UUID, body []byte) (ok bool, log *model.BotEventLog) {
	req, _ := http.NewRequest(http.MethodPost, b.PostURL, bytes.NewReader(body))
	req.Header.Set(headerUserAgent, ua)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
	req.Header.Set(headerTRAQBotEvent, event.String())
	req.Header.Set(headerTRAQBotRequestID, reqID.String())
	req.Header.Set(headerTRAQBotVerificationToken, b.VerificationToken)

	start := time.Now()
	res, err := d.client.Do(req)
	latency := time.Since(start)

	log = &model.BotEventLog{
		RequestID: reqID,
		BotID:     b.ID,
		Event:     event,
		Body:      string(body),
		Latency:   latency.Nanoseconds(),
		DateTime:  start,
	}

	if err != nil {
		eventSendCounter.WithLabelValues(b.ID.String(), resultNetworkError).Inc()
		log.Result = resultNetworkError
		log.Error = err.Error()
		log.Code = -1
		return false, log
	}

	_ = res.Body.Close()
	if res.StatusCode == http.StatusNoContent {
		log.Result = resultOK
	} else {
		log.Result = resultNG
	}
	eventSendCounter.WithLabelValues(b.ID.String(), log.Result).Inc()
	log.Code = res.StatusCode
	return log.Result == resultOK, log
}
