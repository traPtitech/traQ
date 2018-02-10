package notification

import (
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/notification/events"
	"github.com/traPtitech/traQ/router"
	"os"
	"sync"
)

type userId = uuid.UUID
type channelId = uuid.UUID

var (
	notificationStatusCache      map[channelId][]userId
	notificationStatusCacheMutex = sync.RWMutex{}
	streamer                     *sseStreamer
	fcm                          *FCMClient
	isRunning                    = false
	FirebaseServerKey            = os.Getenv("FIREBASE_SERVER_KEY")
)

func Run() {
	if !isRunning {
		isRunning = true
		streamer = &sseStreamer{
			clients:    make(map[uuid.UUID]map[uuid.UUID]*sseClient, 200),
			newConnect: make(chan *sseClient),
			disconnect: make(chan *sseClient, 10),
			stop:       make(chan struct{}),
		}
		fcm = NewFCMClient(FirebaseServerKey)
		go streamer.run()
	}
}

func IsRunning() bool {
	return isRunning
}

func Stop() {
	if isRunning {
		streamer.stop <- struct{}{}
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
		broadcastToAll(&events.EventData{eventType, struct {
			Id string `json:"id"`
		}{data.Id}})

	case events.CHANNEL_CREATED, events.CHANNEL_DELETED, events.CHANNEL_RENAMED, events.CHANNEL_VISIBILITY_CHANGED:
		data, _ := payload.(events.ChannelEvent)
		broadcastToAll(&events.EventData{eventType, struct {
			Id string `json:"id"`
		}{data.Id}})

	case events.CHANNEL_STARED, events.CHANNEL_UNSTARED:
		data, _ := payload.(events.UserStarEvent)
		broadcast(uuid.FromStringOrNil(data.UserId), &events.EventData{eventType, struct {
			Id string `json:"id"`
		}{data.ChannelId}})

	case events.MESSAGE_CREATED, events.MESSAGE_UPDATED, events.MESSAGE_DELETED:
		data, _ := payload.(events.MessageEvent)
		cid := uuid.FromStringOrNil(data.ChannelId)
		done := make(map[userId]bool)
		if s, ok := router.GetHeartbeatStatus(data.ChannelId); ok {
			for _, u := range s.UserStatuses {
				id := uuid.FromStringOrNil(u.UserID)
				if _, ok := done[id]; !ok {
					done[id] = true
					broadcast(id, &events.EventData{eventType, struct {
						Id string `json:"id"`
					}{data.Id}})
				}
			}
		}

		notificationStatusCacheMutex.RLock()
		for _, id := range notificationStatusCache[cid] {
			if _, ok := done[id]; !ok {
				done[id] = true
				broadcast(id, &events.EventData{eventType, struct {
					Id string `json:"id"`
				}{data.Id}})
			}
		}
		defer notificationStatusCacheMutex.RUnlock()

		//TODO ユーザータグによる通知

	case events.MESSAGE_READ:
		data, _ := payload.(events.UserMessageEvent)
		broadcast(uuid.FromStringOrNil(data.UserId), &events.EventData{eventType, struct {
			Id string `json:"id"`
		}{data.MessageId}})

	case events.MESSAGE_STAMPED, events.MESSAGE_UNSTAMPED:
		data, _ := payload.(events.MessageStampEvent)
		if s, ok := router.GetHeartbeatStatus(data.ChannelId); ok {
			for _, u := range s.UserStatuses {
				id := uuid.FromStringOrNil(u.UserID)
				broadcast(id, &events.EventData{eventType, struct {
					Id        string `json:"message_id"`
					ChannelId string `json:"channel_id"`
					UserId    string `json:"user_id"`
					StampId   string `json:"stamp_id"`
					Count     int    `json:"count"`
				}{data.Id, data.ChannelId, data.UserId, data.StampId, data.Count}})
			}
		}

	case events.MESSAGE_PINNED, events.MESSAGE_UNPINNED:
		data, _ := payload.(events.MessageChannelEvent)
		if s, ok := router.GetHeartbeatStatus(data.ChannelId); ok {
			for _, u := range s.UserStatuses {
				broadcast(uuid.FromStringOrNil(u.UserID), &events.EventData{eventType, struct {
					MessageId string `json:"message_id"`
					ChannelId string `json:"channel_id"`
				}{data.MessageId, data.ChannelId}})
			}
		}

	case events.MESSAGE_CLIPPED, events.MESSAGE_UNCLIPPED:
		data, _ := payload.(events.UserMessageEvent)
		broadcast(uuid.FromStringOrNil(data.UserId), &events.EventData{eventType, struct {
			Id string `json:"id"`
		}{data.MessageId}})

	case events.STAMP_CREATED, events.STAMP_DELETED:
		broadcastToAll(&events.EventData{eventType, struct{}{}})

	case events.TRAQ_UPDATED:
		broadcastToAll(&events.EventData{eventType, struct{}{}})
	}
}

func broadcastToAll(data *events.EventData) {
	for _, u := range streamer.clients {
		for _, c := range u {
			c.send <- data
		}
	}
}

func broadcast(target uuid.UUID, data *events.EventData) {
	for _, c := range streamer.clients[target] {
		c.send <- data
	}
}
