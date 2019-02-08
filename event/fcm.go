package event

import (
	"context"
	"strings"
	"time"

	"firebase.google.com/go"
	"firebase.google.com/go/messaging"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/config"
	"github.com/traPtitech/traQ/model"
	"golang.org/x/sync/errgroup"
	"google.golang.org/api/option"
)

// FCMManager Firebaseマネージャー構造体
type FCMManager struct {
	messaging *messaging.Client
}

// FCMEvent FCM通知するイベントのインターフェイス
type FCMEvent interface {
	GetTargetUsers() map[uuid.UUID]bool
	GetExcludeUsers() map[uuid.UUID]bool
	GetFCMData() map[string]string
}

// Init Firebaseサービスを初期化します
func (m *FCMManager) Init() (err error) {
	app, err := firebase.NewApp(context.Background(), nil, option.WithCredentialsFile(config.FirebaseServiceAccountJSONFile))
	if err != nil {
		return err
	}

	m.messaging, err = app.Messaging(context.Background())
	if err != nil {
		return err
	}

	return nil
}

// Process イベントを処理します
func (m *FCMManager) Process(t Type, time time.Time, data interface{}) error {
	e, ok := data.(FCMEvent)
	if !ok {
		return nil
	}

	g := errgroup.Group{}
	payload := e.GetFCMData()
	exclude := e.GetExcludeUsers()
	for u := range e.GetTargetUsers() {
		if exclude[u] {
			continue
		}

		g.Go(func() error {
			devs, err := model.GetDeviceIDs(u)
			if err != nil {
				return err
			}
			return m.sendToFcm(devs, payload)
		})
	}
	return g.Wait()
}

func (m *FCMManager) sendToFcm(deviceTokens []string, data map[string]string) error {
	message := &messaging.Message{
		Data: data,
		Android: &messaging.AndroidConfig{
			Priority: "high",
		},
		APNS: &messaging.APNSConfig{
			Payload: &messaging.APNSPayload{
				Aps: &messaging.Aps{
					Alert: &messaging.ApsAlert{
						Title: data["title"],
						Body:  data["body"],
					},
					Sound:    "default",
					ThreadID: data["tag"],
				},
			},
		},
	}
	for _, token := range deviceTokens {
		message.Token = token
		for i := 0; i < 5; i++ {
			if _, err := m.messaging.Send(context.Background(), message); err != nil {
				switch {
				case strings.Contains(err.Error(), "registration-token-not-registered"):
					fallthrough
				case strings.Contains(err.Error(), "invalid-argument"):
					if err := model.UnregisterDevice(token); err != nil {
						return err
					}
				case strings.Contains(err.Error(), "internal-error"): // 50x
					if i == 4 {
						return err
					}
					continue // リトライ
				default: // 未知のエラー
					return err
				}
			}
			break
		}
	}
	return nil
}
