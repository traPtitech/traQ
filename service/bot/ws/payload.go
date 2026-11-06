package ws

import (
	"net/http"

	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/event"
)

// BotWSConnectedPayload BotWSConnected のペイロード
type BotWSConnectedPayload struct {
	UserID uuid.UUID     `hub:"user_id"`
	Req    *http.Request `hub:"req"`
}

func (BotWSConnectedPayload) TopicName() string { return event.BotWSConnected }

// BotWSDisconnectedPayload BotWSDisconnected のペイロード
type BotWSDisconnectedPayload struct {
	UserID uuid.UUID     `hub:"user_id"`
	Req    *http.Request `hub:"req"`
}

func (BotWSDisconnectedPayload) TopicName() string { return event.BotWSDisconnected }

var (
	_ event.Payload = BotWSConnectedPayload{}
	_ event.Payload = BotWSDisconnectedPayload{}
)
