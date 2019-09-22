package fcm

import (
	"context"
	firebase "firebase.google.com/go"
	"firebase.google.com/go/messaging"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/utils/set"
	"go.uber.org/zap"
	"google.golang.org/api/option"
)

// Client Firebase Cloud Messaging Client
type Client struct {
	c      *messaging.Client
	repo   repository.Repository
	logger *zap.Logger
}

// NewClient Firebase Cloud Messaging Clientを生成します
func NewClient(repo repository.Repository, logger *zap.Logger, options ...option.ClientOption) (*Client, error) {
	app, err := firebase.NewApp(context.Background(), nil, options...)
	if err != nil {
		return nil, err
	}

	mc, err := app.Messaging(context.Background())
	if err != nil {
		return nil, err
	}

	return &Client{
		c:      mc,
		repo:   repo,
		logger: logger,
	}, nil
}

// Send targetユーザーにpayloadを送信します
func (c *Client) Send(targetUserIDs set.UUIDSet, payload *Payload) {
	_ = c.send(targetUserIDs, payload)
}

func (c *Client) send(targetUserIDs set.UUIDSet, payload *Payload) error {
	logger := c.logger.With(zap.Reflect("payload", payload))

	tokens, err := c.repo.GetDeviceTokens(targetUserIDs)
	if err != nil {
		logger.Error("failed to GetDeviceTokens", zap.Error(err), zap.Strings("target_user_ids", targetUserIDs.StringArray()))
		return err
	}
	if len(tokens) == 0 {
		return nil
	}

	var invalidTokens []string
	for _, v := range chunk(tokens, batchSize) {
		ng, err := c.sendOneChunk(v, payload)
		if err != nil {
			logger.Error("an error occurred in sending fcm", zap.Error(err))
			return err
		}
		if len(ng) > 0 {
			invalidTokens = append(invalidTokens, ng...)
		}
	}
	if len(invalidTokens) > 0 {
		err := c.repo.DeleteDeviceTokens(invalidTokens)
		if err != nil {
			logger.Error("failed to DeleteDeviceTokens", zap.Error(err), zap.Strings("invalid_tokens", invalidTokens))
			return err
		}
	}
	return nil
}

func (c *Client) sendOneChunk(tokens []string, payload *Payload) (invalidTokens []string, err error) {
	m := payload.toMessage()
	res, err := c.c.SendMulticast(context.Background(), &messaging.MulticastMessage{
		Tokens:       tokens,
		Data:         m.Data,
		Notification: m.Notification,
		Android:      m.Android,
		Webpush:      m.Webpush,
		APNS:         m.APNS,
	})
	if err != nil {
		return nil, err
	}
	fcmSendCounter.WithLabelValues("error").Add(float64(res.FailureCount))
	fcmSendCounter.WithLabelValues("ok").Add(float64(res.SuccessCount))
	if res.FailureCount > 0 {
		for i, v := range res.Responses {
			if messaging.IsRegistrationTokenNotRegistered(v.Error) {
				invalidTokens = append(invalidTokens, tokens[i])
			}
		}
	}
	return invalidTokens, nil
}
