package fcm

import (
	"context"
	"errors"
	"strconv"
	"time"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"go.uber.org/zap"
	"google.golang.org/api/option"

	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/service/counter"
	"github.com/traPtitech/traQ/service/variable"
	"github.com/traPtitech/traQ/utils/set"
)

type clientImpl struct {
	c             *messaging.Client
	repo          repository.Repository
	logger        *zap.Logger
	unreadCounter counter.UnreadMessageCounter
	queue         chan []*messaging.Message
	close         chan struct{}
}

// NewClientWithCredentialsFile Firebase Cloud Messaging Clientを生成します
func NewClientWithCredentialsFile(repo repository.Repository, logger *zap.Logger, unreadCounter counter.UnreadMessageCounter, file variable.FirebaseCredentialsFilePathString) (Client, error) {
	return NewClient(repo, logger, unreadCounter, option.WithCredentialsFile(string(file)))
}

// NewClient Firebase Cloud Messaging Clientを生成します
func NewClient(repo repository.Repository, logger *zap.Logger, unreadCounter counter.UnreadMessageCounter, options ...option.ClientOption) (Client, error) {
	app, err := firebase.NewApp(context.Background(), nil, options...)
	if err != nil {
		return nil, err
	}

	mc, err := app.Messaging(context.Background())
	if err != nil {
		return nil, err
	}

	c := &clientImpl{
		c:             mc,
		repo:          repo,
		logger:        logger.Named("fcm"),
		unreadCounter: unreadCounter,
		queue:         make(chan []*messaging.Message),
		close:         make(chan struct{}),
	}
	go c.worker()

	return c, nil
}

func (c *clientImpl) Close() {
	if !c.isClosed() {
		close(c.close)
		close(c.queue)
	}
}

func (c *clientImpl) isClosed() bool {
	select {
	case <-c.close:
		return true
	default:
		return false
	}
}

func (c *clientImpl) Send(targetUserIDs set.UUID, payload *Payload, withUnreadCount bool) {
	_ = c.send(targetUserIDs, payload, withUnreadCount)
}

func (c *clientImpl) send(targetUserIDs set.UUID, p *Payload, withUnreadCount bool) error {
	if c.isClosed() {
		return errors.New("fcm client has already been closed")
	}

	logger := c.logger.With(zap.Reflect("payload", p))

	tokensMap, err := c.repo.GetDeviceTokens(targetUserIDs)
	if err != nil {
		logger.Error("failed to GetDeviceTokens", zap.Error(err), zap.Strings("target_user_ids", targetUserIDs.StringArray()))
		return err
	}
	if len(tokensMap) == 0 {
		return nil
	}

	var (
		messages            []*messaging.Message
		apnsHeaders         = map[string]string{"apns-expiration": strconv.FormatInt(time.Now().Add(messageTTL).Unix(), 10)}
		apnsPayloadApsAlert = &messaging.ApsAlert{
			Title: p.Title,
			Body:  p.Body,
		}
	)
	if withUnreadCount {
		for uid, tokens := range tokensMap {
			unread := c.unreadCounter.Get(uid)
			data := map[string]string{
				"type":   p.Type,
				"title":  p.Title,
				"body":   p.Body,
				"path":   p.Path,
				"tag":    p.Tag,
				"icon":   p.Icon,
				"unread": strconv.Itoa(unread),
			}
			if p.Image.Valid {
				data["image"] = p.Image.V
			}
			apns := &messaging.APNSConfig{
				Headers: apnsHeaders,
				Payload: &messaging.APNSPayload{
					Aps: &messaging.Aps{
						Alert:    apnsPayloadApsAlert,
						Sound:    "default",
						ThreadID: p.Tag,
						Badge:    &unread,
					},
				},
			}

			for _, token := range tokens {
				messages = append(messages, &messaging.Message{
					Data:    data,
					Android: defaultAndroidConfig,
					Webpush: defaultWebpushConfig,
					APNS:    apns,
					Token:   token,
				})
			}
		}
	} else {
		data := map[string]string{
			"type":  p.Type,
			"title": p.Title,
			"body":  p.Body,
			"path":  p.Path,
			"tag":   p.Tag,
			"icon":  p.Icon,
		}
		if p.Image.Valid {
			data["image"] = p.Image.V
		}
		apns := &messaging.APNSConfig{
			Headers: apnsHeaders,
			Payload: &messaging.APNSPayload{
				Aps: &messaging.Aps{
					Alert:    apnsPayloadApsAlert,
					Sound:    "default",
					ThreadID: p.Tag,
				},
			},
		}

		for _, tokens := range tokensMap {
			for _, token := range tokens {
				messages = append(messages, &messaging.Message{
					Data:    data,
					Android: defaultAndroidConfig,
					Webpush: defaultWebpushConfig,
					APNS:    apns,
					Token:   token,
				})
			}
		}
	}

	if c.isClosed() {
		return errors.New("fcm client has already been closed")
	}
	c.queue <- messages
	return nil
}

func (c *clientImpl) worker() {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	batch := make([]*messaging.Message, 0, batchSize)
	for {
		select {
		case <-c.close:
			if len(batch) > 0 {
				c.sendMessages(batch)
			}
			return

		case messages := <-c.queue:
			batch = append(batch, messages...)
			if len(batch) >= batchSize {
				go c.sendMessages(batch)
				batch = make([]*messaging.Message, 0, batchSize)
			}

		case <-ticker.C:
			if len(batch) > 0 {
				go c.sendMessages(batch)
				batch = make([]*messaging.Message, 0, batchSize)
			}
		}
	}
}

func (c *clientImpl) sendMessages(messages []*messaging.Message) {
	var invalidTokens []string
	for _, v := range chunkMessages(messages, batchSize) { // 1度に送信できるのは500メッセージまで
		ng, err := c.sendOneChunk(v)
		if err != nil {
			c.logger.Error("an error occurred in sending fcm", zap.Error(err))
			return
		}
		if len(ng) > 0 {
			invalidTokens = append(invalidTokens, ng...)
		}
	}
	if len(invalidTokens) > 0 {
		err := c.repo.DeleteDeviceTokens(invalidTokens)
		if err != nil {
			c.logger.Error("failed to DeleteDeviceTokens", zap.Error(err), zap.Strings("invalid_tokens", invalidTokens))
			return
		}
	}
}

func (c *clientImpl) sendOneChunk(messages []*messaging.Message) (invalidTokens []string, err error) {
	res, err := c.c.SendEach(context.Background(), messages)
	if err != nil {
		fcmBatchRequestCounter.WithLabelValues("error").Inc()
		return nil, err
	}
	fcmBatchRequestCounter.WithLabelValues("ok").Inc()
	fcmSendCounter.WithLabelValues("error").Add(float64(res.FailureCount))
	fcmSendCounter.WithLabelValues("ok").Add(float64(res.SuccessCount))
	if res.FailureCount > 0 {
		for i, v := range res.Responses {
			if v.Error == nil {
				continue
			}
			switch {
			case messaging.IsUnregistered(v.Error):
				invalidTokens = append(invalidTokens, messages[i].Token)
			default:
				c.logger.Warn("fcm: "+v.Error.Error(), zap.String("token", messages[i].Token))
			}
		}
	}
	return invalidTokens, nil
}

func chunkMessages(s []*messaging.Message, size int) (r [][]*messaging.Message) {
	for size < len(s) {
		s, r = s[size:], append(r, s[0:size:size])
	}
	return append(r, s)
}
