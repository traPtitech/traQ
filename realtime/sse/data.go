package sse

import (
	jsoniter "github.com/json-iterator/go"
	"net/http"
)

// EventData SSEイベントデータ
type EventData struct {
	EventType string
	Payload   interface{}
}

func (d *EventData) write(rw http.ResponseWriter) {
	stream := jsoniter.ConfigFastest.BorrowStream(rw)
	_, _ = rw.Write([]byte("event: "))
	_, _ = rw.Write([]byte(d.EventType))
	_, _ = rw.Write([]byte("\ndata: "))
	stream.WriteVal(d.Payload)
	_ = stream.Flush()
	jsoniter.ConfigFastest.ReturnStream(stream)
	_, _ = rw.Write([]byte("\n\n"))
}
