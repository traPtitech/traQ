package ws

import "github.com/gofrs/uuid"

type rawMessage struct {
	t    int
	data []byte
}

type marshalledRaw []byte

func (m marshalledRaw) MarshalJSON() ([]byte, error) {
	return m, nil
}

type eventMessage struct {
	Type  string        `json:"type"`
	ReqID uuid.UUID     `json:"reqId"`
	Body  marshalledRaw `json:"body"`
}

func makeEventMessage(t string, reqID uuid.UUID, b []byte) (m *eventMessage) {
	return &eventMessage{
		Type:  t,
		ReqID: reqID,
		Body:  b,
	}
}

func (m *eventMessage) toJSON() (b []byte) {
	b, _ = json.Marshal(m)
	return
}

type errorMessage struct {
	Type string      `json:"type"`
	Body interface{} `json:"body"`
}

func makeErrorMessage(b interface{}) (m *errorMessage) {
	return &errorMessage{
		Type: "ERROR",
		Body: b,
	}
}

func (m *errorMessage) toJSON() (b []byte) {
	b, _ = json.Marshal(m)
	return
}
