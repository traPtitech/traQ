package sse

import (
	"encoding/json"
	"net/http"
)

// EventData SSEイベントデータ
type EventData struct {
	EventType string
	Payload   interface{}
}

func (d *EventData) write(rw http.ResponseWriter) {
	data, _ := json.Marshal(d.Payload)
	_, _ = rw.Write([]byte("event: "))
	_, _ = rw.Write([]byte(d.EventType))
	_, _ = rw.Write([]byte("\ndata: "))
	_, _ = rw.Write(data)
	_, _ = rw.Write([]byte("\n\n"))
}
