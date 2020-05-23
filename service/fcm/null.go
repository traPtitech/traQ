package fcm

import (
	"github.com/traPtitech/traQ/utils/set"
)

var nullC = &nullClient{}

type nullClient struct{}

// NewNullClient 何もしないFCMクライアントを返します
func NewNullClient() Client {
	return nullC
}

func (n *nullClient) Send(set.UUID, *Payload, bool) {
	return
}

func (n *nullClient) Close() {
	return
}
