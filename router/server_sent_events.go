package router

import (
	"encoding/json"
	"github.com/labstack/echo"
	"github.com/leandro-lugaresi/hub"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/utils/message"
	"net/http"
	"sync"
	"time"
)

// Payload データペイロード型
type Payload map[string]interface{}

type eventData struct {
	EventType string
	Payload   Payload
}

// SSEStreamer SSEストリーマー
type SSEStreamer struct {
	sseClientMap
	repo       repository.Repository
	connect    chan *sseClient
	disconnect chan *sseClient
	stop       chan struct{}
}

type sseClientMap struct {
	sync.Map
}

func (m *sseClientMap) loadClients(key uuid.UUID) (map[uuid.UUID]*sseClient, bool) {
	i, ok := m.Load(key)
	if ok {
		return i.(map[uuid.UUID]*sseClient), true
	}
	return nil, false
}

func (m *sseClientMap) storeClients(key uuid.UUID, value map[uuid.UUID]*sseClient) {
	m.Store(key, value)
}

func (m *sseClientMap) rangeClients(f func(key uuid.UUID, value map[uuid.UUID]*sseClient) bool) {
	m.Range(func(k, v interface{}) bool {
		return f(k.(uuid.UUID), v.(map[uuid.UUID]*sseClient))
	})
}

func (m *sseClientMap) broadcast(data *eventData) {
	m.rangeClients(func(_ uuid.UUID, u map[uuid.UUID]*sseClient) bool {
		for _, c := range u {
			c.RLock()
			skip := c.disconnected
			c.RUnlock()
			if skip {
				continue
			}

			c.send <- data
		}
		return true
	})
}

func (m *sseClientMap) multicast(user uuid.UUID, data *eventData) {
	if u, ok := m.loadClients(user); ok {
		for _, c := range u {
			c.RLock()
			skip := c.disconnected
			c.RUnlock()
			if skip {
				continue
			}

			c.send <- data
		}
	}
}

type sseClient struct {
	sync.RWMutex
	userID       uuid.UUID
	connectionID uuid.UUID
	send         chan *eventData
	disconnected bool
}

func (c *sseClient) dispose() {
	c.Lock()
	c.disconnected = true
	c.Unlock()
	close(c.send)
	// flush buffer
	for range c.send {
	}
}

// NewSSEStreamer SSEストリーマーを作成します
func NewSSEStreamer(hub *hub.Hub, repo repository.Repository) *SSEStreamer {
	s := &SSEStreamer{
		repo:       repo,
		connect:    make(chan *sseClient),
		disconnect: make(chan *sseClient, 10),
		stop:       make(chan struct{}),
	}
	go func() {
		for {
			select {
			case <-s.stop:
				close(s.connect)
				close(s.disconnect)
				return

			case c := <-s.connect:
				arr, ok := s.loadClients(c.userID)
				if !ok {
					arr = make(map[uuid.UUID]*sseClient)
					s.storeClients(c.userID, arr)
				}
				arr[c.connectionID] = c

			case c := <-s.disconnect:
				arr, _ := s.loadClients(c.userID)
				delete(arr, c.connectionID)
			}
		}
	}()
	s.setupSubscriber(hub)
	return s
}

// Dispose SSEストリーマーを破棄します
func (s *SSEStreamer) Dispose() {
	close(s.stop)
}

func (s *SSEStreamer) setupSubscriber(h *hub.Hub) {
	go func(sub hub.Subscription) {
		for ev := range sub.Receiver {
			m := ev.Fields["message"].(*model.Message)
			p := ev.Fields["plain"].(string)
			e := ev.Fields["embedded"].([]*message.EmbeddedInfo)
			go s.processMessageCreated(m, p, e)
		}
	}(h.Subscribe(10,
		event.MessageCreated,
	))

	go func(sub hub.Subscription) {
		for ev := range sub.Receiver {
			private := ev.Fields["private"].(bool)
			if private {
				go s.processUserMulticastEvent(ev)
			} else {
				go s.processBroadcastEvent(ev)
			}
		}
	}(h.Subscribe(10,
		event.ChannelCreated,
		event.ChannelDeleted,
		event.ChannelUpdated,
	))

	go func(sub hub.Subscription) {
		for ev := range sub.Receiver {
			go s.processChannelUserMulticastEvent(ev)
		}
	}(h.Subscribe(10,
		event.MessageUpdated,
		event.MessageDeleted,
		event.MessagePinned,
		event.MessageUnpinned,
		event.MessageStamped,
		event.MessageUnstamped,
	))

	go func(sub hub.Subscription) {
		for ev := range sub.Receiver {
			go s.processUserMulticastEvent(ev)
		}
	}(h.Subscribe(10,
		event.ChannelStared,
		event.ChannelUnstared,
		event.ChannelMuted,
		event.ChannelUnmuted,
		event.ClipCreated,
		event.ClipDeleted,
		event.ClipMoved,
		event.ClipFolderCreated,
		event.ClipFolderUpdated,
		event.ClipFolderDeleted,
		event.ChannelRead,
	))

	go func(sub hub.Subscription) {
		for ev := range sub.Receiver {
			go s.processBroadcastEvent(ev)
		}
	}(h.Subscribe(10,
		event.UserCreated,
		event.UserUpdated,
		event.UserIconUpdated,
		event.UserOnline,
		event.UserOffline,
		event.UserTagAdded,
		event.UserTagUpdated,
		event.UserTagRemoved,
		event.UserGroupCreated,
		event.UserGroupDeleted,
		event.UserGroupMemberAdded,
		event.UserGroupMemberRemoved,
		event.StampCreated,
		event.StampUpdated,
		event.StampDeleted,
	))
}

func (s *SSEStreamer) processMessageCreated(message *model.Message, plain string, embedded []*message.EmbeddedInfo) {
	ed := &eventData{
		EventType: "MESSAGE_CREATED",
		Payload: Payload{
			"id": message.ID,
		},
	}
	viewers := map[uuid.UUID]bool{}
	connector := map[uuid.UUID]bool{}
	subscribers := map[uuid.UUID]bool{}
	ch, _ := s.repo.GetChannel(message.ChannelID)
	switch {
	case ch.IsForced: // 強制通知チャンネル
		users, _ := s.repo.GetUsers()
		for _, v := range users {
			if v.Bot {
				continue
			}
			subscribers[v.ID] = true
		}

	case !ch.IsPublic: // プライベートチャンネル
		users, _ := s.repo.GetPrivateChannelMemberIDs(ch.ID)
		for _, v := range users {
			subscribers[v] = true
		}

	default: // 通常チャンネルメッセージ
		// チャンネル通知ユーザー取得
		users, _ := s.repo.GetSubscribingUserIDs(message.ChannelID)
		for _, v := range users {
			subscribers[v] = true
		}

		// タグユーザー・メンションユーザー取得
		for _, v := range embedded {
			switch v.Type {
			case "user":
				if uid, err := uuid.FromString(v.ID); err != nil {
					subscribers[uid] = true
				}
			case "tag":
				tagged, _ := s.repo.GetUserIDsByTagID(uuid.FromStringOrNil(v.ID))
				for _, v := range tagged {
					subscribers[v] = true
				}
			}
		}
	}

	// ハートビートユーザー取得
	if s, ok := s.repo.GetHeartbeatStatus(message.ChannelID); ok {
		for _, u := range s.UserStatuses {
			connector[u.UserID] = true
			if u.Status != "none" {
				viewers[u.UserID] = true
			}
		}
	}

	// 送信
	for id := range subscribers {
		if !(id == message.UserID || viewers[id]) {
			_ = s.repo.SetMessageUnread(id, message.ID)
		}
		go s.multicast(id, ed)
	}
	for id := range connector {
		if !subscribers[id] {
			go s.multicast(id, ed)
		}
	}
}

func (s *SSEStreamer) processUserMulticastEvent(ev hub.Message) {
	var (
		ed      *eventData
		targets = map[uuid.UUID]bool{}
	)
	switch ev.Topic() {
	case event.ChannelCreated:
		cid := ev.Fields["channel_id"].(uuid.UUID)
		ed = &eventData{
			EventType: "CHANNEL_CREATED",
			Payload: Payload{
				"id": cid,
			},
		}
		members, _ := s.repo.GetPrivateChannelMemberIDs(cid)
		for _, u := range members {
			targets[u] = true
		}
	case event.ChannelUpdated:
		cid := ev.Fields["channel_id"].(uuid.UUID)
		ed = &eventData{
			EventType: "CHANNEL_UPDATED",
			Payload: Payload{
				"id": cid,
			},
		}
		members, _ := s.repo.GetPrivateChannelMemberIDs(cid)
		for _, u := range members {
			targets[u] = true
		}
	case event.ChannelDeleted:
		cid := ev.Fields["channel_id"].(uuid.UUID)
		ed = &eventData{
			EventType: "CHANNEL_DELETED",
			Payload: Payload{
				"id": cid,
			},
		}
		members, _ := s.repo.GetPrivateChannelMemberIDs(cid)
		for _, u := range members {
			targets[u] = true
		}
	case event.ChannelStared:
		ed = &eventData{
			EventType: "CHANNEL_STARED",
			Payload: Payload{
				"id": ev.Fields["channel_id"].(uuid.UUID),
			},
		}
		targets[ev.Fields["user_id"].(uuid.UUID)] = true
	case event.ChannelUnstared:
		ed = &eventData{
			EventType: "CHANNEL_UNSTARED",
			Payload: Payload{
				"id": ev.Fields["channel_id"].(uuid.UUID),
			},
		}
		targets[ev.Fields["user_id"].(uuid.UUID)] = true
	case event.ChannelMuted:
		ed = &eventData{
			EventType: "CHANNEL_MUTED",
			Payload: Payload{
				"id": ev.Fields["channel_id"].(uuid.UUID),
			},
		}
		targets[ev.Fields["user_id"].(uuid.UUID)] = true
	case event.ChannelUnmuted:
		ed = &eventData{
			EventType: "CHANNEL_UNMUTED",
			Payload: Payload{
				"id": ev.Fields["channel_id"].(uuid.UUID),
			},
		}
		targets[ev.Fields["user_id"].(uuid.UUID)] = true
	case event.ClipCreated:
		ed = &eventData{
			EventType: "CLIP_CREATED",
			Payload: Payload{
				"id": ev.Fields["clip_id"].(uuid.UUID),
			},
		}
		targets[ev.Fields["user_id"].(uuid.UUID)] = true
	case event.ClipDeleted:
		ed = &eventData{
			EventType: "CLIP_DELETED",
			Payload: Payload{
				"id": ev.Fields["clip_id"].(uuid.UUID),
			},
		}
		targets[ev.Fields["user_id"].(uuid.UUID)] = true
	case event.ClipMoved:
		ed = &eventData{
			EventType: "CLIP_MOVED",
			Payload: Payload{
				"id": ev.Fields["clip_id"].(uuid.UUID),
			},
		}
		targets[ev.Fields["user_id"].(uuid.UUID)] = true
	case event.ClipFolderCreated:
		ed = &eventData{
			EventType: "CLIP_FOLDER_CREATED",
			Payload: Payload{
				"id": ev.Fields["folder_id"].(uuid.UUID),
			},
		}
		targets[ev.Fields["user_id"].(uuid.UUID)] = true
	case event.ClipFolderUpdated:
		ed = &eventData{
			EventType: "CLIP_FOLDER_UPDATED",
			Payload: Payload{
				"id": ev.Fields["folder_id"].(uuid.UUID),
			},
		}
		targets[ev.Fields["user_id"].(uuid.UUID)] = true
	case event.ClipFolderDeleted:
		ed = &eventData{
			EventType: "CLIP_FOLDER_DELETED",
			Payload: Payload{
				"id": ev.Fields["folder_id"].(uuid.UUID),
			},
		}
		targets[ev.Fields["user_id"].(uuid.UUID)] = true
	case event.ChannelRead:
		ed = &eventData{
			EventType: "MESSAGE_READ",
			Payload: Payload{
				"id": ev.Fields["channel_id"].(uuid.UUID),
			},
		}
		targets[ev.Fields["user_id"].(uuid.UUID)] = true
	}
	for u := range targets {
		go s.multicast(u, ed)
	}
}

func (s *SSEStreamer) processChannelUserMulticastEvent(ev hub.Message) {
	var (
		ed  *eventData
		cid uuid.UUID
	)
	switch ev.Topic() {
	case event.MessageUpdated:
		ed = &eventData{
			EventType: "MESSAGE_UPDATED",
			Payload: Payload{
				"id": ev.Fields["message_id"].(uuid.UUID),
			},
		}
		cid = ev.Fields["message"].(*model.Message).ChannelID
	case event.MessageDeleted:
		ed = &eventData{
			EventType: "MESSAGE_DELETED",
			Payload: Payload{
				"id": ev.Fields["message_id"].(uuid.UUID),
			},
		}
		cid = ev.Fields["message"].(*model.Message).ChannelID
	case event.MessagePinned:
		ed = &eventData{
			EventType: "MESSAGE_PINNED",
			Payload: Payload{
				"id": ev.Fields["pin_id"].(uuid.UUID),
			},
		}
		ch, err := s.repo.GetChannelByMessageID(ev.Fields["message_id"].(uuid.UUID))
		if err != nil {
			return
		}
		cid = ch.ID
	case event.MessageUnpinned:
		ed = &eventData{
			EventType: "MESSAGE_UNPINNED",
			Payload: Payload{
				"id": ev.Fields["pin_id"].(uuid.UUID),
			},
		}
		ch, err := s.repo.GetChannelByMessageID(ev.Fields["message_id"].(uuid.UUID))
		if err != nil {
			return
		}
		cid = ch.ID
	case event.MessageStamped:
		ed = &eventData{
			EventType: "MESSAGE_STAMPED",
			Payload: Payload{
				"message_id": ev.Fields["message_id"].(uuid.UUID),
				"user_id":    ev.Fields["user_id"].(uuid.UUID),
				"stamp_id":   ev.Fields["stamp_id"].(uuid.UUID),
				"count":      ev.Fields["count"].(int),
				"created_at": ev.Fields["created_at"].(time.Time),
			},
		}
		ch, err := s.repo.GetChannelByMessageID(ev.Fields["message_id"].(uuid.UUID))
		if err != nil {
			return
		}
		cid = ch.ID
	case event.MessageUnstamped:
		ed = &eventData{
			EventType: "MESSAGE_UNSTAMPED",
			Payload: Payload{
				"message_id": ev.Fields["message_id"].(uuid.UUID),
				"user_id":    ev.Fields["user_id"].(uuid.UUID),
				"stamp_id":   ev.Fields["stamp_id"].(uuid.UUID),
			},
		}
		ch, err := s.repo.GetChannelByMessageID(ev.Fields["message_id"].(uuid.UUID))
		if err != nil {
			return
		}
		cid = ch.ID
	}
	if status, ok := s.repo.GetHeartbeatStatus(cid); ok {
		for _, u := range status.UserStatuses {
			go s.multicast(u.UserID, ed)
		}
	}
}

func (s *SSEStreamer) processBroadcastEvent(ev hub.Message) {
	var ed *eventData
	switch ev.Topic() {
	case event.ChannelCreated:
		ed = &eventData{
			EventType: "CHANNEL_CREATED",
			Payload: Payload{
				"id": ev.Fields["channel_id"].(uuid.UUID),
			},
		}
	case event.ChannelUpdated:
		ed = &eventData{
			EventType: "CHANNEL_UPDATED",
			Payload: Payload{
				"id": ev.Fields["channel_id"].(uuid.UUID),
			},
		}
	case event.ChannelDeleted:
		ed = &eventData{
			EventType: "CHANNEL_DELETED",
			Payload: Payload{
				"id": ev.Fields["channel_id"].(uuid.UUID),
			},
		}
	case event.UserCreated:
		ed = &eventData{
			EventType: "USER_JOINED",
			Payload: Payload{
				"id": ev.Fields["user_id"].(uuid.UUID),
			},
		}
	case event.UserUpdated:
		ed = &eventData{
			EventType: "USER_UPDATED",
			Payload: Payload{
				"id": ev.Fields["user_id"].(uuid.UUID),
			},
		}
	case event.UserIconUpdated:
		ed = &eventData{
			EventType: "USER_ICON_UPDATED",
			Payload: Payload{
				"id": ev.Fields["user_id"].(uuid.UUID),
			},
		}
	case event.UserOnline:
		ed = &eventData{
			EventType: "USER_ONLINE",
			Payload: Payload{
				"id": ev.Fields["user_id"].(uuid.UUID),
			},
		}
	case event.UserOffline:
		ed = &eventData{
			EventType: "USER_OFFLINE",
			Payload: Payload{
				"id": ev.Fields["user_id"].(uuid.UUID),
			},
		}
	case event.UserTagAdded, event.UserTagUpdated, event.UserTagRemoved:
		ed = &eventData{
			EventType: "USER_TAGS_UPDATED",
			Payload: Payload{
				"id": ev.Fields["user_id"].(uuid.UUID),
			},
		}
	case event.UserGroupCreated:
		ed = &eventData{
			EventType: "USER_GROUP_CREATED",
			Payload: Payload{
				"id": ev.Fields["group_id"].(uuid.UUID),
			},
		}
	case event.UserGroupDeleted:
		ed = &eventData{
			EventType: "USER_GROUP_DELETED",
			Payload: Payload{
				"id": ev.Fields["group_id"].(uuid.UUID),
			},
		}
	case event.UserGroupMemberAdded:
		ed = &eventData{
			EventType: "USER_GROUP_MEMBER_ADDED",
			Payload: Payload{
				"id":      ev.Fields["group_id"].(uuid.UUID),
				"user_id": ev.Fields["user_id"].(uuid.UUID),
			},
		}
	case event.UserGroupMemberRemoved:
		ed = &eventData{
			EventType: "USER_GROUP_MEMBER_REMOVED",
			Payload: Payload{
				"id":      ev.Fields["group_id"].(uuid.UUID),
				"user_id": ev.Fields["user_id"].(uuid.UUID),
			},
		}
	case event.StampCreated:
		ed = &eventData{
			EventType: "STAMP_CREATED",
			Payload: Payload{
				"id": ev.Fields["stamp_id"].(uuid.UUID),
			},
		}
	case event.StampUpdated:
		ed = &eventData{
			EventType: "STAMP_MODIFIED",
			Payload: Payload{
				"id": ev.Fields["stamp_id"].(uuid.UUID),
			},
		}
	case event.StampDeleted:
		ed = &eventData{
			EventType: "STAMP_DELETED",
			Payload: Payload{
				"id": ev.Fields["stamp_id"].(uuid.UUID),
			},
		}
	}
	go s.broadcast(ed)
}

// NotificationStream GET /notification
func (h *Handlers) NotificationStream(c echo.Context) error {
	if _, ok := c.Response().Writer.(http.Flusher); !ok {
		return echo.NewHTTPError(http.StatusNotImplemented, "Server Sent Events is not supported.")
	}

	//Set headers for SSE
	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")
	c.Response().Header().Set("X-Accel-Buffering", "no") // for nginx
	c.Response().WriteHeader(http.StatusOK)

	client := &sseClient{
		userID:       c.Get("user").(*model.User).ID,
		connectionID: uuid.NewV4(),
		send:         make(chan *eventData, 100),
	}
	h.SSE.connect <- client

	res := c.Response()
	rw := res.Writer
	fl := rw.(http.Flusher)
	cn := res.CloseNotify()

	t := time.NewTicker(10 * time.Second)
	defer t.Stop()

	fl.Flush()
StreamFor:
	for {
		select {
		case <-h.SSE.stop: // サーバーが停止
			client.dispose()
			break StreamFor

		case <-cn: // クライアントが切断
			client.dispose()
			h.SSE.disconnect <- client
			break StreamFor

		case m := <-client.send: // イベントを送信
			data, _ := json.Marshal(m.Payload)
			_, _ = rw.Write([]byte("event: "))
			_, _ = rw.Write([]byte(m.EventType))
			_, _ = rw.Write([]byte("\ndata: "))
			_, _ = rw.Write(data)
			_, _ = rw.Write([]byte("\n\n"))
			fl.Flush()

		case <-t.C: // タイムアウト対策で10秒おきにコメント行を送信する
			_, _ = rw.Write([]byte(":\n\n"))
			fl.Flush()
		}
	}

	return nil
}
