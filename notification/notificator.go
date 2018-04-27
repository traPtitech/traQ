package notification

import (
	"context"
	"firebase.google.com/go"
	"firebase.google.com/go/messaging"
	"github.com/labstack/gommon/log"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/config"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/notification/events"
	"github.com/traPtitech/traQ/utils/message"
	"google.golang.org/api/option"
	"strings"
)

type eventData struct {
	EventType events.EventType
	Payload   events.DataPayload
}

var (
	streamer  *sseStreamer
	isStarted = false
	fcm       *messaging.Client
)

//Start 通知機構を起動します
func Start() {
	if !isStarted {
		isStarted = true
		streamer = newSseStreamer()
		if len(config.FirebaseServiceAccountJSONFile) > 0 {
			app, err := firebase.NewApp(context.Background(), nil, option.WithCredentialsFile(config.FirebaseServiceAccountJSONFile))
			if err != nil {
				panic(err)
			}
			m, err := app.Messaging(context.Background())
			if err != nil {
				panic(err)
			}
			fcm = m
		}
		go streamer.run()
	}
}

//IsStarted 通知機構が起動しているかどうかを返します
func IsStarted() bool {
	return isStarted
}

//Stop 通知機構を停止します
func Stop() {
	if isStarted {
		close(streamer.stop)
		isStarted = false
	}
}

//Send 通知イベントを発行します
func Send(eventType events.EventType, payload interface{}) {
	if !isStarted {
		return
	}

	switch eventType {
	case events.MessageCreated:
		data := payload.(events.MessageEvent)
		cid := data.TargetChannel()
		viewers := map[uuid.UUID]bool{}
		connector := map[uuid.UUID]bool{}
		subscribers := map[uuid.UUID]bool{}

		if ch, err := model.GetChannelByMessageID(data.Message.ID); err != nil {
			log.Error(err)
		} else if ch.IsForced {
			// 強制通知
			users, err := model.GetUsers()
			if err != nil {
				log.Error(err)
			}
			for _, v := range users {
				if v.Bot {
					continue
				}
				subscribers[uuid.FromStringOrNil(v.ID)] = true
			}
		} else if !ch.IsPublic {
			// 強制通知(プライベートチャンネル)
			users, err := model.GetPrivateChannelMembers(ch.ID)
			if err != nil {
				log.Error(err)
			}
			for _, v := range users {
				subscribers[uuid.FromStringOrNil(v)] = true
			}
		} else {
			// チャンネル通知ユーザー取得
			if users, err := model.GetSubscribingUser(cid); err != nil {
				log.Error(err)
			} else {
				for _, v := range users {
					subscribers[v] = true
				}
			}

			// タグユーザー・メンションユーザー取得
			ei, _ := message.Parse(data.Message.Text)
			for _, v := range ei {
				switch v.Type {
				case "user":
					subscribers[uuid.FromStringOrNil(v.ID)] = true
				case "tag":
					if users, err := model.GetUserIDsByTagID(v.ID); err != nil {
						log.Error(err)
					} else {
						for _, v := range users {
							subscribers[uuid.FromStringOrNil(v)] = true
						}
					}
				}
			}

		}

		// ハートビートユーザー取得
		if s, ok := model.GetHeartbeatStatus(cid.String()); ok {
			for _, u := range s.UserStatuses {
				connector[uuid.FromStringOrNil(u.UserID)] = true
				if u.Status != "none" {
					viewers[uuid.FromStringOrNil(u.UserID)] = true
				}
			}
		}

		// 送信
		eData := &eventData{
			EventType: eventType,
			Payload:   data.DataPayload(),
		}
		mPayload := data.GetData()
		for id := range subscribers {
			if id.String() == data.Message.UserID || viewers[id] {
				go multicast(id, eData)
			} else {
				if err := (&model.Unread{UserID: id.String(), MessageID: data.Message.ID}).Create(); err != nil {
					log.Error(err)
				}

				go multicast(id, eData)
				go func() {
					devs, err := model.GetDeviceIDs(id)
					if err != nil {
						log.Error(err)
						return
					}
					sendToFcm(devs, mPayload)
				}()
			}
		}

		for id := range connector {
			if !subscribers[id] {
				go multicast(id, eData)
			}
		}

	default:
		switch e := payload.(type) {
		case events.UserTargetEvent: // ユーザーマルチキャストイベント
			go multicast(e.TargetUser(), &eventData{
				EventType: eventType,
				Payload:   e.DataPayload(),
			})

		case events.ChannelUserTargetEvent: // チャンネルユーザーマルチキャストイベント
			data := &eventData{
				EventType: eventType,
				Payload:   e.DataPayload(),
			}
			if s, ok := model.GetHeartbeatStatus(e.TargetChannel().String()); ok {
				for _, u := range s.UserStatuses {
					go multicast(uuid.FromStringOrNil(u.UserID), data)
				}
			}

		case events.Event: // ブロードキャストイベント
			go broadcast(&eventData{
				EventType: eventType,
				Payload:   e.DataPayload(),
			})
		}
	}
}

func broadcast(data *eventData) {
	streamer.clients.Range(func(_ uuid.UUID, u map[uuid.UUID]*sseClient) bool {
		for _, c := range u {
			select {
			case <-c.stop:
				continue
			default:
				c.send <- data
			}
		}
		return true
	})
}

func multicast(target uuid.UUID, data *eventData) {
	if u, ok := streamer.clients.Load(target); ok {
		for _, c := range u {
			select {
			case <-c.stop:
				continue
			default:
				c.send <- data
			}
		}
	}
}

func sendToFcm(deviceTokens []string, data map[string]string) {
	m := &messaging.Message{
		Data: data,
		Android: &messaging.AndroidConfig{
			Priority: "high",
		},
	}
	for _, token := range deviceTokens {
		m.Token = token
		if _, err := fcm.Send(context.Background(), m); err != nil {
			switch {
			case strings.Contains(err.Error(), "registration-token-not-registered"):
				fallthrough
			case strings.Contains(err.Error(), "invalid-argument"):
				if err := (&model.Device{Token: token}).Unregister(); err != nil {
					log.Error(err)
				}
			default:
				//TODO loggingを真面目にする
				log.Error(err)
			}
		}
	}
}
