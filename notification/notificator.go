package notification

import (
	"github.com/labstack/gommon/log"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/notification/events"
	"os"
)

var (
	streamer          *sseStreamer
	fcm               *fcmClient
	isRunning         = false
	FirebaseServerKey = os.Getenv("FIREBASE_SERVER_KEY")
)

func Run() {
	if !isRunning {
		isRunning = true
		streamer = newSseStreamer()
		fcm = newFCMClient(FirebaseServerKey)
		go streamer.run()
	}
}

func IsRunning() bool {
	return isRunning
}

func Stop() {
	if isRunning {
		close(streamer.stop)
		fcm = nil
		isRunning = false
	}
}

func Send(eventType events.EventType, payload interface{}) {
	if !isRunning {
		return
	}

	switch eventType {
	case events.UserJoined, events.UserLeft, events.UserTagsUpdated:
		data, _ := payload.(events.UserEvent)
		broadcastToAll(&events.EventData{
			EventType: eventType,
			Payload: struct {
				Id string `json:"id"`
			}{data.Id},
			Mobile: false,
		})

	case events.ChannelCreated, events.ChannelDeleted, events.ChannelUpdated, events.ChannelVisibilityChanged:
		data, _ := payload.(events.ChannelEvent)
		broadcastToAll(&events.EventData{
			EventType: eventType,
			Payload: struct {
				Id string `json:"id"`
			}{data.Id},
			Mobile: false,
		})

	case events.ChannelStared, events.ChannelUnstared:
		data, _ := payload.(events.UserChannelEvent)
		broadcast(uuid.FromStringOrNil(data.UserId), &events.EventData{
			EventType: eventType,
			Payload: struct {
				Id string `json:"id"`
			}{data.ChannelId},
			Mobile: false,
		})

	case events.MessageCreated, events.MessageUpdated, events.MessageDeleted:
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
				broadcast(id, &events.EventData{
					EventType: eventType,
					Summary:   "", //TODO モバイル通知に表示される文字列
					Payload: struct {
						Id string `json:"id"`
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
					broadcast(id, &events.EventData{
						EventType: eventType,
						Summary:   "", //TODO モバイル通知に表示される文字列
						Payload: struct {
							Id string `json:"id"`
						}{data.Message.ID},
						Mobile: true,
					})
				}
			}
		}

		if len(tags) > 0 {
			if users, err := model.GetUserIdsByTags(tags); err != nil {
				log.Error(err)
			} else {
				for _, id := range users {
					if _, ok := done[id]; !ok {
						done[id] = true
						broadcast(id, &events.EventData{
							EventType: eventType,
							Summary:   "", //TODO モバイル通知に表示される文字列
							Payload: struct {
								Id string `json:"id"`
							}{data.Message.ID},
							Mobile: true,
						})
					}
				}
			}
		}

	case events.MessageRead:
		data, _ := payload.(events.UserMessageEvent)
		broadcast(uuid.FromStringOrNil(data.UserId), &events.EventData{
			EventType: eventType,
			Payload: struct {
				Id string `json:"id"`
			}{data.MessageId},
			Mobile: false,
		})

	case events.MessageStamped, events.MessageUnstamped:
		data, _ := payload.(events.MessageStampEvent)
		if s, ok := model.GetHeartbeatStatus(data.ChannelId); ok {
			for _, u := range s.UserStatuses {
				id := uuid.FromStringOrNil(u.UserID)
				broadcast(id, &events.EventData{
					EventType: eventType,
					Payload: struct {
						Id        string `json:"message_id"`
						ChannelId string `json:"channel_id"`
						UserId    string `json:"user_id"`
						StampId   string `json:"stamp_id"`
						Count     int    `json:"count"`
					}{data.Id, data.ChannelId, data.UserId, data.StampId, data.Count},
					Mobile: false,
				})
			}
		}

	case events.MessagePinned, events.MessageUnpinned:
		data, _ := payload.(events.MessageChannelEvent)
		if s, ok := model.GetHeartbeatStatus(data.ChannelId); ok {
			for _, u := range s.UserStatuses {
				broadcast(uuid.FromStringOrNil(u.UserID), &events.EventData{
					EventType: eventType,
					Payload: struct {
						MessageId string `json:"message_id"`
						ChannelId string `json:"channel_id"`
					}{data.MessageId, data.ChannelId},
					Mobile: false,
				})
			}
		}

	case events.MessageClipped, events.MessageUnclipped:
		data, _ := payload.(events.UserMessageEvent)
		broadcast(uuid.FromStringOrNil(data.UserId), &events.EventData{
			EventType: eventType,
			Payload: struct {
				Id string `json:"id"`
			}{data.MessageId},
			Mobile: false,
		})

	case events.StampCreated, events.StampDeleted:
		broadcastToAll(&events.EventData{
			EventType: eventType,
			Payload:   struct{}{},
			Mobile:    false,
		})

	case events.TraqUpdated:
		broadcastToAll(&events.EventData{
			EventType: eventType,
			Payload:   struct{}{},
			Mobile:    false,
		})
	}
}

func broadcastToAll(data *events.EventData) {
	for _, u := range streamer.clients {
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
		devs, err := model.GetAllDeviceIds()
		if err != nil {
			log.Error(err)
			return
		}
		broadcastToFcm(devs, data)
	}
}

func broadcast(target uuid.UUID, data *events.EventData) {
	for _, c := range streamer.clients[target] {
		select {
		case <-c.stop:
			continue
		default:
			c.send <- data
		}
	}

	if data.Mobile {
		devs, err := model.GetDeviceIds(target)
		if err != nil {
			log.Error(err)
			return
		}
		broadcastToFcm(devs, data)
	}
}

func broadcastToFcm(deviceTokens []string, data *events.EventData) {
	for arr := range split(deviceTokens, maxRegistrationIdsSize) {
		m := createDefaultFCMMessage()
		m.Notification.Body = data.Summary
		m.Data = data.Payload
		m.RegistrationIds = arr

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
