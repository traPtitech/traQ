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
	fcm               *FCMClient
	isRunning         = false
	FirebaseServerKey = os.Getenv("FIREBASE_SERVER_KEY")
)

func Run() {
	if !isRunning {
		isRunning = true
		streamer = NewSseStreamer()
		fcm = NewFCMClient(FirebaseServerKey)
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
	case events.USER_JOINED, events.USER_LEFT, events.USER_TAGS_UPDATED:
		data, _ := payload.(events.UserEvent)
		broadcastToAll(&events.EventData{
			EventType: eventType,
			Payload: struct {
				Id string `json:"id"`
			}{data.Id},
			Mobile: false,
		})

	case events.CHANNEL_CREATED, events.CHANNEL_DELETED, events.CHANNEL_RENAMED, events.CHANNEL_VISIBILITY_CHANGED:
		data, _ := payload.(events.ChannelEvent)
		broadcastToAll(&events.EventData{
			EventType: eventType,
			Payload: struct {
				Id string `json:"id"`
			}{data.Id},
			Mobile: false,
		})

	case events.CHANNEL_STARED, events.CHANNEL_UNSTARED:
		data, _ := payload.(events.UserStarEvent)
		broadcast(uuid.FromStringOrNil(data.UserId), &events.EventData{
			EventType: eventType,
			Payload: struct {
				Id string `json:"id"`
			}{data.ChannelId},
			Mobile: false,
		})

	case events.MESSAGE_CREATED, events.MESSAGE_UPDATED, events.MESSAGE_DELETED:
		data, _ := payload.(events.MessageEvent)
		cid := uuid.FromStringOrNil(data.ChannelId)
		done := make(map[uuid.UUID]bool)

		//MEMO 通知ユーザー・ユーザータグのキャッシュを使ったほうがいいかもしれない。
		if users, err := model.GetSubscribingUser(cid); err != nil {
			log.Error(err)
		} else {
			for _, id := range users {
				done[id] = true
				broadcast(id, &events.EventData{
					EventType: eventType,
					Summary:   "", //TODO
					Payload: struct {
						//TODO
						Id string `json:"id"`
					}{data.Id},
					Mobile: true,
				})
			}
		}

		if s, ok := model.GetHeartbeatStatus(data.ChannelId); ok {
			for _, u := range s.UserStatuses {
				id := uuid.FromStringOrNil(u.UserID)
				if _, ok := done[id]; !ok {
					done[id] = true
					broadcast(id, &events.EventData{
						EventType: eventType,
						Summary:   "", //TODO
						Payload: struct {
							//TODO
							Id string `json:"id"`
						}{data.Id},
						Mobile: true,
					})
				}
			}
		}

		if len(data.Tags) > 0 {
			if users, err := model.GetUserIdsByTags(data.Tags); err != nil {
				log.Error(err)
			} else {
				for _, id := range users {
					if _, ok := done[id]; !ok {
						done[id] = true
						broadcast(id, &events.EventData{
							EventType: eventType,
							Summary:   "", //TODO
							Payload: struct {
								//TODO
								Id string `json:"id"`
							}{data.Id},
							Mobile: true,
						})
					}
				}
			}
		}

	case events.MESSAGE_READ:
		data, _ := payload.(events.UserMessageEvent)
		broadcast(uuid.FromStringOrNil(data.UserId), &events.EventData{
			EventType: eventType,
			Payload: struct {
				Id string `json:"id"`
			}{data.MessageId},
			Mobile: false,
		})

	case events.MESSAGE_STAMPED, events.MESSAGE_UNSTAMPED:
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

	case events.MESSAGE_PINNED, events.MESSAGE_UNPINNED:
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

	case events.MESSAGE_CLIPPED, events.MESSAGE_UNCLIPPED:
		data, _ := payload.(events.UserMessageEvent)
		broadcast(uuid.FromStringOrNil(data.UserId), &events.EventData{
			EventType: eventType,
			Payload: struct {
				Id string `json:"id"`
			}{data.MessageId},
			Mobile: false,
		})

	case events.STAMP_CREATED, events.STAMP_DELETED:
		broadcastToAll(&events.EventData{
			EventType: eventType,
			Payload:   struct{}{},
			Mobile:    false,
		})

	case events.TRAQ_UPDATED:
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
	for arr := range split(deviceTokens, MaxRegistrationIdsSize) {
		m := createDefaultFCMMessage()
		m.Notification.Body = data.Summary
		m.Data = data.Payload
		m.RegistrationIds = arr

		res, err := fcm.Send(m)
		if err != nil {
			log.Error(err)
			continue
		}
		if res.IsTimeout() {
			//TODO Retry
		} else if res.Failure > 0 {
			for _, t := range res.GetInvalidRegistration() {
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

func createDefaultFCMMessage() *FCMMessage {
	return &FCMMessage{
		Notification:     createDefaultFCMNotificationPayload(),
		Priority:         PriorityHigh,
		ContentAvailable: true,
		DryRun:           false,
	}
}

func createDefaultFCMNotificationPayload() *FCMNotificationPayload {
	return &FCMNotificationPayload{
		Title: "traQ",
	}
}
