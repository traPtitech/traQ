package ws

import "github.com/traPtitech/traQ/utils/jsonutil"

type rawMessage struct {
	t    int
	data []byte
}

type message struct {
	Type string      `json:"type"`
	Body interface{} `json:"body"`
}

func makeMessage(t string, b interface{}) (m *message) {
	return &message{
		Type: t,
		Body: b,
	}
}

func (m *message) toJSON() (b []byte) {
	b, _ = jsonutil.Marshal(m)
	return
}
