package notification

import (
	"github.com/labstack/gommon/log"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/notification/events"
	"os"
	"time"
)

type eventData struct {
	EventType events.EventType
	Summary   string
	Payload   interface{}
	Mobile    bool
}

var (
	streamer                       *sseStreamer
	fcm                            *fcmClient
	isStarted                      = false
	firebaseServiceAccountJSONFile = os.Getenv("FIREBASE_SERVICE_ACCOUNT_JSON")
)

//Start 通知機構を起動します
func Start() {
	if !isStarted {
		isStarted = true
		streamer = newSseStreamer()
		fcm = newFCMClient(firebaseServiceAccountJSONFile)
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
		fcm = nil
		isStarted = false
	}
}

//Send 通知イベントを発行します
func Send(eventType events.EventType, payload interface{}) {
	if !isStarted {
		return
	}

	switch eventType {
	case events.UserJoined, events.UserLeft, events.UserUpdated, events.UserTagsUpdated, events.UserIconUpdated:
		data, _ := payload.(events.UserEvent)
		multicastToAll(&eventData{
			EventType: eventType,
			Payload: struct {
				ID string `json:"id"`
			}{data.ID},
			Mobile: false,
		})

	case events.ChannelCreated, events.ChannelDeleted, events.ChannelUpdated, events.ChannelVisibilityChanged:
		data, _ := payload.(events.ChannelEvent)
		multicastToAll(&eventData{
			EventType: eventType,
			Payload: struct {
				ID string `json:"id"`
			}{data.ID},
			Mobile: false,
		})

	case events.ChannelStared, events.ChannelUnstared:
		data, _ := payload.(events.UserChannelEvent)
		multicast(uuid.FromStringOrNil(data.UserID), &eventData{
			EventType: eventType,
			Payload: struct {
				ID string `json:"id"`
			}{data.ChannelID},
			Mobile: false,
		})

	case events.MessageCreated:
		data, _ := payload.(events.MessageEvent)
		cid := uuid.FromStringOrNil(data.Message.ChannelID)
		var tags []string //TODO タグ抽出
		done := make(map[uuid.UUID]bool)

		//MEMO 通知ユーザー・ユーザータグのキャッシュを使ったほうがいいかもしれない。
		if users, err := model.GetSubscribingUser(cid); err != nil {
			log.Error(err)
		} else {
			for _, id := range users {
				done[id] = true

				unread := &model.Unread{UserID: id.String(), MessageID: data.Message.ID}
				if err := unread.Create(); err != nil {
					log.Error(err)
				}

				multicast(id, &eventData{
					EventType: eventType,
					Summary:   "", //TODO モバイル通知に表示される文字列
					Payload: struct {
						ID string `json:"id"`
					}{data.Message.ID},
					Mobile: true,
				})
			}
		}

		if s, ok := model.GetHeartbeatStatus(data.Message.ChannelID); ok {
			for _, u := range s.UserStatuses {
				id := uuid.FromStringOrNil(u.UserID)
				if _, ok := done[id]; !ok {
					done[id] = true

					unread := &model.Unread{UserID: id.String(), MessageID: data.Message.ID}
					if err := unread.Create(); err != nil {
						log.Error(err)
					}

					multicast(id, &eventData{
						EventType: eventType,
						Summary:   "", //TODO モバイル通知に表示される文字列
						Payload: struct {
							ID string `json:"id"`
						}{data.Message.ID},
						Mobile: true,
					})
				}
			}
		}

		if len(tags) > 0 {
			if users, err := model.GetUserIDsByTags(tags); err != nil {
				log.Error(err)
			} else {
				for _, id := range users {
					if _, ok := done[id]; !ok {
						done[id] = true

						unread := &model.Unread{UserID: id.String(), MessageID: data.Message.ID}
						if err := unread.Create(); err != nil {
							log.Error(err)
						}

						multicast(id, &eventData{
							EventType: eventType,
							Summary:   "", //TODO モバイル通知に表示される文字列
							Payload: struct {
								ID string `json:"id"`
							}{data.Message.ID},
							Mobile: true,
						})
					}
				}
			}
		}

	case events.MessageUpdated, events.MessageDeleted:
		data, _ := payload.(events.MessageEvent)

		if s, ok := model.GetHeartbeatStatus(data.Message.ChannelID); ok {
			for _, u := range s.UserStatuses {
				id := uuid.FromStringOrNil(u.UserID)
				multicast(id, &eventData{
					EventType: eventType,
					Payload: struct {
						ID string `json:"id"`
					}{data.Message.ID},
					Mobile: false,
				})
			}
		}

	case events.MessageRead:
		data, _ := payload.(events.ReadMessagesEvent)
		multicast(uuid.FromStringOrNil(data.UserID), &eventData{
			EventType: eventType,
			Payload: struct {
				IDs []string `json:"ids"`
			}{data.MessageIDs},
			Mobile: false,
		})

	case events.MessageStamped:
		data, _ := payload.(events.MessageStampEvent)
		if s, ok := model.GetHeartbeatStatus(data.ChannelID); ok {
			for _, u := range s.UserStatuses {
				id := uuid.FromStringOrNil(u.UserID)
				multicast(id, &eventData{
					EventType: eventType,
					Payload: struct {
						ID        string    `json:"message_id"`
						UserID    string    `json:"user_id"`
						StampID   string    `json:"stamp_id"`
						Count     int       `json:"count"`
						CreatedAt time.Time `json:"created_at"`
					}{data.ID, data.UserID, data.StampID, data.Count, data.CreatedAt},
					Mobile: false,
				})
			}
		}

	case events.MessageUnstamped:
		data, _ := payload.(events.MessageStampEvent)
		if s, ok := model.GetHeartbeatStatus(data.ChannelID); ok {
			for _, u := range s.UserStatuses {
				id := uuid.FromStringOrNil(u.UserID)
				multicast(id, &eventData{
					EventType: eventType,
					Payload: struct {
						ID      string `json:"message_id"`
						UserID  string `json:"user_id"`
						StampID string `json:"stamp_id"`
					}{data.ID, data.UserID, data.StampID},
					Mobile: false,
				})
			}
		}
	case events.MessagePinned, events.MessageUnpinned:
		data, _ := payload.(events.PinEvent)
		if s, ok := model.GetHeartbeatStatus(data.Message.ChannelID); ok {
			for _, u := range s.UserStatuses {
				multicast(uuid.FromStringOrNil(u.UserID), &eventData{
					EventType: eventType,
					Payload: struct {
						ID string `json:"id"`
					}{data.PinID},
					Mobile: false,
				})
			}
		}

	case events.MessageClipped, events.MessageUnclipped:
		data, _ := payload.(events.UserMessageEvent)
		multicast(uuid.FromStringOrNil(data.UserID), &eventData{
			EventType: eventType,
			Payload: struct {
				ID string `json:"id"`
			}{data.MessageID},
			Mobile: false,
		})

	case events.StampCreated, events.StampModified, events.StampDeleted:
		data, _ := payload.(events.StampEvent)
		multicastToAll(&eventData{
			EventType: eventType,
			Payload: struct {
				ID string `json:"id"`
			}{data.ID},
			Mobile: false,
		})

	case events.TraqUpdated:
		multicastToAll(&eventData{
			EventType: eventType,
			Payload:   struct{}{},
			Mobile:    false,
		})
	}
}

func multicastToAll(data *eventData) {
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
		sendToFcm(devs, data)
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

	if data.Mobile {
		devs, err := model.GetDeviceIDs(target)
		if err != nil {
			log.Error(err)
			return
		}
		sendToFcm(devs, data)
	}
}

func sendToFcm(deviceTokens []string, data *eventData) {
	for arr := range split(deviceTokens, maxRegistrationIdsSize) {
		m := createDefaultFCMMessage()
		m.Notification.Body = data.Summary
		m.Data = data.Payload
		m.RegistrationIDs = arr

		res, err := fcm.send(m)
		if err != nil {
			log.Error(err)
			continue
		}
		if res.isTimeout() {
			//TODO Retry
		} else if res.Failure > 0 {
			for _, t := range res.getInvalidRegistration() {
				device := &model.Device{Token: t}
				if err := device.Unregister(); err != nil {
					log.Error(err)
				}
			}
		}
	}
}

func split(dev []string, n int) chan []string {
	ch := make(chan []string)

	go func() {
		for i := 0; i < len(dev); i += n {
			from := i
			to := i + n
			if to > len(dev) {
				to = len(dev)
			}
			ch <- dev[from:to]
		}
		close(ch)
	}()
	return ch
}

func createDefaultFCMMessage() *fcmMessage {
	return &fcmMessage{
		Notification:     createDefaultFCMNotificationPayload(),
		Priority:         priorityHigh,
		ContentAvailable: true,
		DryRun:           false,
	}
}

func createDefaultFCMNotificationPayload() *fcmNotificationPayload {
	return &fcmNotificationPayload{
		Title: "traQ",
	}
}
