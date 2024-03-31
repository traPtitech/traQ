package fcm

import (
	"strconv"
	"time"

	"firebase.google.com/go/v4/messaging"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"golang.org/x/exp/utf8string"

	"github.com/traPtitech/traQ/utils/optional"
)

const (
	batchSize            = 500
	messageTTLSeconds    = 60 * 60 * 24 * 2 // 2日
	notificationPriority = "high"
)

var (
	fcmSendCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "firebase",
		Name:      "fcm_send_count_total",
	}, []string{"result"})
	fcmBatchRequestCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "firebase",
		Name:      "fcm_batch_request_count_total",
	}, []string{"result"})
	messageTTL       = messageTTLSeconds * time.Second
	messageTTLString = strconv.Itoa(messageTTLSeconds)

	defaultAndroidConfig = &messaging.AndroidConfig{
		Priority: notificationPriority,
		TTL:      &messageTTL,
	}
	defaultWebpushConfig = &messaging.WebpushConfig{
		Headers: map[string]string{
			"TTL": messageTTLString,
		},
	}
)

// Payload FCMペイロード
type Payload struct {
	Type  string
	Title string
	Body  string
	Icon  string
	Path  string
	Tag   string
	Image optional.Of[string]
}

// SetBodyWithEllipsis 100文字を超える場合は...で省略
func (p *Payload) SetBodyWithEllipsis(body string) {
	if s := utf8string.NewString(body); s.RuneCount() > 100 {
		body = s.Slice(0, 100) + "..."
	}
	p.Body = body
}
