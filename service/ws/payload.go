package ws

import (
	"net/http"

	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/event"
)

// WSConnectedPayload WSConnected のペイロード
type WSConnectedPayload struct { //nolint:revive // あとで fix したいから #<PR番号> で暫定的に入れている
	UserID uuid.UUID     `hub:"user_id"`
	Req    *http.Request `hub:"req"`
}

func (WSConnectedPayload) TopicName() string { return event.WSConnected }

// WSDisconnectedPayload WSDisconnected のペイロード
type WSDisconnectedPayload struct { //nolint:revive // あとで fix したいから #<PR番号> で暫定的に入れている
	UserID uuid.UUID     `hub:"user_id"`
	Req    *http.Request `hub:"req"`
}

func (WSDisconnectedPayload) TopicName() string { return event.WSDisconnected }

var (
	_ event.Payload = WSConnectedPayload{}
	_ event.Payload = WSDisconnectedPayload{}
)
