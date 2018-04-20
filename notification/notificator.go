package notification

import (
	"context"
	"firebase.google.com/go"
	"firebase.google.com/go/messaging"
	"fmt"
	"github.com/labstack/gommon/log"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/config"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/notification/events"
	"github.com/traPtitech/traQ/utils/message"
	"golang.org/x/exp/utf8string"
	"google.golang.org/api/option"
	"strings"
)

type eventData struct {
	EventType events.EventType
	Summary   string
	Payload   events.DataPayload
	Mobile    bool
	IconURL   string
	Action    string
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
		summary := ""

		ei, plain := message.Parse(data.Message.Text)
		path, _ := model.GetChannelPath(cid)
		action := fmt.Sprintf("%s/channels/%s", config.TRAQOrigin, strings.TrimPrefix(path, "#"))
		user, _ := model.GetUser(data.Message.UserID)
		if user == nil {
			user = &model.User{DisplayName: "ERROR"}
		} else if len(user.DisplayName) == 0 {
			user.DisplayName = user.Name
		}

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

			summary = fmt.Sprintf("[%s] %s: %s", path, user.DisplayName, plain)
		} else if !ch.IsPublic {
			// 強制通知(プライベートチャンネル)
			users, err := model.GetPrivateChannelMembers(ch.ID)
			if err != nil {
				log.Error(err)
			}
			for _, v := range users {
				subscribers[uuid.FromStringOrNil(v)] = true
			}

			if l := len(users); l == 2 || l == 1 {
				// DM
				action = fmt.Sprintf("%s/users/%s", config.TRAQOrigin, user.Name)
				summary = fmt.Sprintf("[@%s] %s", user.Name, plain)
			} else {
				// Private Channel
				summary = fmt.Sprintf("[%s] %s: %s", path, user.DisplayName, plain)
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

			summary = fmt.Sprintf("[%s] %s: %s", path, user.DisplayName, plain)
		}

		if s := utf8string.NewString(summary); s.RuneCount() > 100 {
			summary = s.Slice(0, 97) + "..."
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
		for id := range subscribers {
			if id.String() == data.Message.UserID {
				multicast(id, &eventData{
					EventType: eventType,
					Summary:   summary,
					Payload:   data.DataPayload(),
					Mobile:    false,
				})
			} else {
				if viewers[id] {
					multicast(id, &eventData{
						EventType: eventType,
						Summary:   summary,
						Payload:   data.DataPayload(),
						Mobile:    false,
					})
				} else {
					// 未読リストに追加
					unread := &model.Unread{UserID: id.String(), MessageID: data.Message.ID}
					if err := unread.Create(); err != nil {
						log.Error(err)
					}
					multicast(id, &eventData{
						EventType: eventType,
						Summary:   summary,
						Payload:   data.DataPayload(),
						Mobile:    true,
						IconURL:   fmt.Sprintf("%s/api/1.0/users/%s/icon?thumb", config.TRAQOrigin, data.Message.UserID),
						Action:    action,
					})
				}
			}
		}

		for id := range connector {
			if !subscribers[id] {
				multicast(id, &eventData{
					EventType: eventType,
					Summary:   summary,
					Payload:   data.DataPayload(),
					Mobile:    false,
				})
			}
		}

	default:
		switch e := payload.(type) {
		case events.UserTargetEvent: // ユーザーマルチキャストイベント
			multicast(e.TargetUser(), &eventData{
				EventType: eventType,
				Payload:   e.DataPayload(),
				Mobile:    false,
			})

		case events.ChannelUserTargetEvent: // チャンネルユーザーマルチキャストイベント
			if s, ok := model.GetHeartbeatStatus(e.TargetChannel().String()); ok {
				for _, u := range s.UserStatuses {
					multicast(uuid.FromStringOrNil(u.UserID), &eventData{
						EventType: eventType,
						Payload:   e.DataPayload(),
						Mobile:    false,
					})
				}
			}

		case events.Event: // ブロードキャストイベント
			broadcast(&eventData{
				EventType: eventType,
				Payload:   e.DataPayload(),
				Mobile:    false,
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

	if data.Mobile {
		devs, err := model.GetAllDeviceIDs()
		if err != nil {
			log.Error(err)
			return
		}
		sendToFcm(devs, data.Summary, data.Payload, data.IconURL, data.Action)
	}
}

func multicast(target uuid.UUID, data *eventData) {
	u, ok := streamer.clients.Load(target)
	if ok {
		for _, c := range u {
			select {
			case <-c.stop:
				continue
			default:
				c.send <- data
			}
		}
	}

	if data.Mobile && fcm != nil {
		devs, err := model.GetDeviceIDs(target)
		if err != nil {
			log.Error(err)
			return
		}
		sendToFcm(devs, data.Summary, data.Payload, data.IconURL, data.Action)
	}
}

func sendToFcm(deviceTokens []string, body string, payload events.DataPayload, iconURL, action string) {
	data := map[string]string{
		"origin": config.TRAQOrigin,
	}
	if len(action) > 0 {
		data["click_action"] = action
	}
	for k, v := range payload {
		switch t := v.(type) {
		case fmt.Stringer:
			data[k] = t.String()
		default:
			data[k] = fmt.Sprint(t)
		}
	}

	m := &messaging.Message{
		Data: data,
		Notification: &messaging.Notification{
			Title: "traQ",
			Body:  body,
		},
		Android: &messaging.AndroidConfig{
			Priority: "high",
			Notification: &messaging.AndroidNotification{
				Icon:        iconURL,
				ClickAction: action,
			},
		},
		Webpush: &messaging.WebpushConfig{
			Notification: &messaging.WebpushNotification{
				Icon: iconURL,
			},
		},
	}
	for _, token := range deviceTokens {
		m.Token = token

		_, err := fcm.Send(context.Background(), m)
		if err != nil {
			switch {
			case strings.Contains(err.Error(), "registration-token-not-registered"):
				fallthrough
			case strings.Contains(err.Error(), "invalid-argument"):
				device := &model.Device{Token: token}
				if err := device.Unregister(); err != nil {
					log.Error(err)
				}
			default:
				//TODO loggingを真面目にする
				log.Error(err)
			}
		}
	}
}
