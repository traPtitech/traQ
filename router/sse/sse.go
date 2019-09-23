package sse

import (
	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"net/http"
	"time"
)

var sseConnectionsCounter = promauto.NewGauge(prometheus.GaugeOpts{
	Namespace: "traq",
	Name:      "sse_connections",
})

// Streamer SSEストリーマー
type Streamer struct {
	sseClientMap
	connect    chan *sseClient
	disconnect chan *sseClient
	stop       chan struct{}
}

// NewStreamer SSEストリーマーを作成します
func NewStreamer() *Streamer {
	s := &Streamer{
		connect:    make(chan *sseClient),
		disconnect: make(chan *sseClient, 10),
		stop:       make(chan struct{}),
	}
	go func() {
		for {
			select {
			case <-s.stop:
				close(s.connect)
				close(s.disconnect)
				return

			case c := <-s.connect:
				arr, ok := s.loadClients(c.userID)
				if !ok {
					arr = make(map[uuid.UUID]*sseClient)
					s.storeClients(c.userID, arr)
				}
				arr[c.connectionID] = c

			case c := <-s.disconnect:
				arr, _ := s.loadClients(c.userID)
				delete(arr, c.connectionID)
			}
		}
	}()
	return s
}

// Dispose SSEストリーマーを破棄します
func (s *Streamer) Dispose() {
	close(s.stop)
}

// Broadcast イベントデータを全コネクションに配信します
func (s *Streamer) Broadcast(data *EventData) {
	s.broadcast(data)
}

// Multicast イベントデータを指定ユーザーの全コネクションに配信します
func (s *Streamer) Multicast(userID uuid.UUID, data *EventData) {
	s.multicast(userID, data)
}

// ServeHTTP http.Handlerインターフェイスの実装
func (s *Streamer) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Set("Content-Type", "text/event-stream")
	rw.Header().Set("Cache-Control", "no-cache, no-transform")
	rw.Header().Set("Connection", "keep-alive")
	rw.Header().Set("X-Accel-Buffering", "no") // for nginx
	rw.WriteHeader(http.StatusOK)

	ctx := r.Context()
	client := &sseClient{
		userID:       ctx.Value(CtxUserIDKey).(uuid.UUID),
		connectionID: uuid.Must(uuid.NewV4()),
		send:         make(chan *EventData, 100),
	}
	s.connect <- client

	sseConnectionsCounter.Inc()
	defer sseConnectionsCounter.Dec()

	t := time.NewTicker(10 * time.Second)
	defer t.Stop()

	fl := rw.(http.Flusher)
	fl.Flush()
StreamFor:
	for {
		select {
		case <-s.stop: // サーバーが停止
			client.dispose()
			break StreamFor

		case <-ctx.Done(): // クライアントが切断
			client.dispose()
			s.disconnect <- client
			break StreamFor

		case m := <-client.send: // イベントを送信
			m.write(rw)
			fl.Flush()

		case <-t.C: // タイムアウト対策で10秒おきにコメント行を送信する
			_, _ = rw.Write([]byte(":\n\n"))
			fl.Flush()
		}
	}
}
