package fcm

import (
	"firebase.google.com/go/messaging"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/traPtitech/traQ/utils/optional"
	"golang.org/x/exp/utf8string"
	"strconv"
	"time"
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
	Image optional.String
}

// SetBodyWithEllipsis 100文字を超える場合は...で省略
func (p *Payload) SetBodyWithEllipsis(body string) {
	if s := utf8string.NewString(body); s.RuneCount() > 100 {
		body = s.Slice(0, 100) + "..."
	}
	p.Body = body
}

func (p *Payload) toMessage() *messaging.Message {
	data := map[string]string{
		"type":  p.Type,
		"title": p.Title,
		"body":  p.Body,
		"path":  p.Path,
		"tag":   p.Tag,
		"icon":  p.Icon,
	}
	if p.Image.Valid {
		data["image"] = p.Image.String
	}

	return &messaging.Message{
		// データ メッセージとして全て処理する
		Data:    data,
		Android: defaultAndroidConfig,
		Webpush: defaultWebpushConfig,
		APNS: &messaging.APNSConfig{
			Headers: map[string]string{
				"apns-expiration": strconv.FormatInt(time.Now().Add(messageTTL).Unix(), 10),
			},
			Payload: &messaging.APNSPayload{
				Aps: &messaging.Aps{
					Alert: &messaging.ApsAlert{
						Title: p.Title,
						Body:  p.Body,
					},
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
