package fcm

import (
	"firebase.google.com/go/messaging"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"golang.org/x/exp/utf8string"
	"strconv"
	"time"
)

const (
	batchSize            = 100
	messageTTLSeconds    = 60 * 60 * 24 * 2 // 2日
	notificationPriority = "high"
)

var (
	fcmSendCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "firebase",
		Name:      "fcm_send_count_total",
	}, []string{"result"})
	messageTTL       = messageTTLSeconds * time.Second
	messageTTLString = strconv.Itoa(messageTTLSeconds)
)

// Payload FCMペイロード
type Payload struct {
	Title string
	Body  string
	Icon  string
	Path  string
	Tag   string
	Image string
}

// SetBodyWithEllipsis 100文字を超える場合は...で省略
func (p *Payload) SetBodyWithEllipsis(body string) {
	if s := utf8string.NewString(body); s.RuneCount() > 100 {
		body = s.Slice(0, 100) + "..."
	}
	p.Body = body
}

func (p *Payload) toMessage() *messaging.Message {
	return &messaging.Message{
		// データ メッセージとして全て処理する
		Data: map[string]string{
			"title": p.Title,
			"body":  p.Body,
			"path":  p.Path,
			"tag":   p.Tag,
			"icon":  p.Icon,
			"image": p.Image,
		},
		Android: &messaging.AndroidConfig{
			Priority: notificationPriority,
			TTL:      &messageTTL,
		},
		Webpush: &messaging.WebpushConfig{
			Headers: map[string]string{
				"TTL": messageTTLString,
			},
		},
		APNS: &messaging.APNSConfig{
			Headers: map[string]string{
				"apns-expiration": strconv.FormatInt(time.Now().Add(messageTTL).Unix(), 10),
			},
			Payload: &messaging.APNSPayload{
				Aps: &messaging.Aps{
					Sound:    "default",
					ThreadID: p.Tag,
				},
			},
		},
	}
}

func chunk(s []string, size int) (r [][]string) {
	for size < len(s) {
		s, r = s[size:], append(r, s[0:size:size])
	}
	return append(r, s)
}
