package ws

import (
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait          = 5 * time.Second
	pongWait           = 60 * time.Second
	pingPeriod         = (pongWait * 9) / 10
	maxReadMessageSize = 1 << 9 // 512B
	messageBufferSize  = 256
)

var (
	upgrader = &websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(_ *http.Request) bool { return true },
	}
)
