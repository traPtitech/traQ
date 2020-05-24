package fcm

import "github.com/traPtitech/traQ/utils/set"

// Client Firebase Cloud Messaging Client
type Client interface {
	// Send targetユーザーにpayloadを送信します
	Send(targetUserIDs set.UUID, payload *Payload, withUnreadCount bool)
	Close()
}
