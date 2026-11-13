package event

import (
	"net/http"

	"github.com/gofrs/uuid"
)

// WSConnectedPayload WSConnected のペイロード
type WSConnectedPayload struct {
	UserID uuid.UUID     `hub:"user_id"`
	Req    *http.Request `hub:"req"`
}

func (WSConnectedPayload) TopicName() string { return WSConnected }

// WSDisconnectedPayload WSDisconnected のペイロード
type WSDisconnectedPayload struct {
	UserID uuid.UUID     `hub:"user_id"`
	Req    *http.Request `hub:"req"`
}

func (WSDisconnectedPayload) TopicName() string { return WSDisconnected }

// BotWSConnectedPayload BotWSConnected のペイロード
type BotWSConnectedPayload struct {
	UserID uuid.UUID     `hub:"user_id"`
	Req    *http.Request `hub:"req"`
}

func (BotWSConnectedPayload) TopicName() string { return BotWSConnected }

// BotWSDisconnectedPayload BotWSDisconnected のペイロード
type BotWSDisconnectedPayload struct {
	UserID uuid.UUID     `hub:"user_id"`
	Req    *http.Request `hub:"req"`
}

func (BotWSDisconnectedPayload) TopicName() string { return BotWSDisconnected }

var (
	_ Payload = WSConnectedPayload{}
	_ Payload = WSDisconnectedPayload{}
	_ Payload = BotWSConnectedPayload{}
	_ Payload = BotWSDisconnectedPayload{}
)
