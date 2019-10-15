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
	data, _ := jsoniter.ConfigCompatibleWithStandardLibrary.Marshal(d.Payload)
	_, _ = rw.Write([]byte("event: "))
	_, _ = rw.Write([]byte(d.EventType))
	_, _ = rw.Write([]byte("\ndata: "))
	_, _ = rw.Write(data)
	_, _ = rw.Write([]byte("\n\n"))
}

// CtxKey context.Context用のキータイプ
type CtxKey int

const (
	// CtxUserIDKey ユーザーUUIDキー
	CtxUserIDKey CtxKey = iota
)
