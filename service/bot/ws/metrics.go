package ws

import (
	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	webSocketReadBytesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "traq",
		Name:      "bot_ws_read_bytes_total",
	}, []string{"user_id"})

	webSocketWriteBytesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "traq",
		Name:      "bot_ws_write_bytes_total",
	}, []string{"user_id"})
)

func incWebSocketReadBytesTotal(userID uuid.UUID, bytes int) {
	webSocketReadBytesTotal.WithLabelValues(userID.String()).Add(float64(bytes))
}

func incWebSocketWriteBytesTotal(userID uuid.UUID, bytes int) {
	webSocketWriteBytesTotal.WithLabelValues(userID.String()).Add(float64(bytes))
}
