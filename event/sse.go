package event

import (
	"encoding/json"
	"github.com/labstack/echo"
	"github.com/labstack/gommon/log"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/model"
	"net/http"
	"sync"
	"time"
)

var sseSeparator = []byte("\n\n")

// SSEEvent SSEイベントのインターフェイス
type SSEEvent interface {
	GetData() Payload
}

// UsersTargetEvent 特定のユーザー宛のイベントのインターフェイス
type UsersTargetEvent interface {
	SSEEvent
	GetTargetUsers() map[uuid.UUID]bool
}

// ChannelViewersTargetEvent 特定のチャンネルを見ているユーザー宛のイベントのインターフェイス
type ChannelViewersTargetEvent interface {
	SSEEvent
	GetTargetChannels() map[uuid.UUID]bool
}

// SSEStreamer SSEストリーマー
type SSEStreamer struct {
	clients    sync.Map
	connect    chan *sseClient
	disconnect chan *sseClient
	stop       chan struct{}
}

type eventData struct {
	EventType Type
	Payload   Payload
}

type sseClient struct {
	userID       uuid.UUID
	connectionID uuid.UUID
	send         chan *eventData
	stop         chan struct{}
}

// NewSSEStreamer SSEストリーマーを作成します
func NewSSEStreamer() *SSEStreamer {
	streamer := &SSEStreamer{
		connect:    make(chan *sseClient),
		disconnect: make(chan *sseClient, 10),
		stop:       make(chan struct{}),
	}
	go streamer.loop()

	return streamer
}

func (s *SSEStreamer) loadClients(key uuid.UUID) (map[uuid.UUID]*sseClient, bool) {
	i, ok := s.clients.Load(key)
	if ok {
		return i.(map[uuid.UUID]*sseClient), true
	}
	return nil, false
}

func (s *SSEStreamer) storeClients(key uuid.UUID, value map[uuid.UUID]*sseClient) {
	s.clients.Store(key, value)
}

func (s *SSEStreamer) rangeClients(f func(key uuid.UUID, value map[uuid.UUID]*sseClient) bool) {
	s.clients.Range(func(k, v interface{}) bool {
		return f(k.(uuid.UUID), v.(map[uuid.UUID]*sseClient))
	})
}

func (s *SSEStreamer) loop() {
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
}

// Dispose SSEストリーマーを破棄します
func (s *SSEStreamer) Dispose() {
	close(s.stop)
}

// Process イベントを処理し、SSEを送信します
func (s *SSEStreamer) Process(t Type, time time.Time, data interface{}) error {
	ev, ok := data.(SSEEvent)
	if !ok {
		return nil
	}

	ed := &eventData{
		EventType: t,
		Payload:   ev.GetData(),
	}
	switch t {
	case MessageCreated:
		me := data.(*MessageCreatedEvent)
		cid := uuid.Must(uuid.FromString(me.Message.ChannelID))
		viewers := map[uuid.UUID]bool{}
		connector := map[uuid.UUID]bool{}

		// 通知対象のユーザーを取得
		subscribers := me.GetTargetUsers()

		// ハートビートユーザー取得
		if s, ok := model.GetHeartbeatStatus(cid.String()); ok {
			for _, u := range s.UserStatuses {
				connector[uuid.Must(uuid.FromString(u.UserID))] = true
				if u.Status != "none" {
					viewers[uuid.Must(uuid.FromString(u.UserID))] = true
				}
			}
		}

		// 送信
		for id := range subscribers {
			if !(id.String() == me.Message.UserID || viewers[id]) {
				if err := model.SetMessageUnread(id, me.Message.GetID()); err != nil {
					log.Error(err)
				}
			}
			go s.multicast(id, ed)
		}
		for id := range connector {
			if !subscribers[id] {
				go s.multicast(id, ed)
			}
		}

	default:
		switch e := data.(type) {
		case UsersTargetEvent: // ユーザーマルチキャストイベント
			for u := range e.GetTargetUsers() {
				go s.multicast(u, ed)
			}

		case ChannelViewersTargetEvent: // チャンネルユーザーマルチキャストイベント
			for c := range e.GetTargetChannels() {
				if status, ok := model.GetHeartbeatStatus(c.String()); ok {
					for _, u := range status.UserStatuses {
						go s.multicast(uuid.Must(uuid.FromString(u.UserID)), ed)
					}
				}
			}

		default: // ブロードキャストイベント
			go s.broadcast(ed)
		}
	}

	return nil
}

func (s *SSEStreamer) broadcast(data *eventData) {
	s.rangeClients(func(_ uuid.UUID, u map[uuid.UUID]*sseClient) bool {
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

func (s *SSEStreamer) multicast(user uuid.UUID, data *eventData) {
	if u, ok := s.loadClients(user); ok {
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

// StreamHandler Server-Sent EventsのHTTPハンドラ
func (s *SSEStreamer) StreamHandler(c echo.Context) error {
	if _, ok := c.Response().Writer.(http.Flusher); !ok {
		return echo.NewHTTPError(http.StatusNotImplemented, "Server Sent Events is not supported.")
	}

	//Set headers for SSE
	c.Response().Header().Set(echo.HeaderContentType, "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")
	c.Response().WriteHeader(http.StatusOK)

	client := &sseClient{
		userID:       c.Get("user").(*model.User).GetUID(),
		connectionID: uuid.NewV4(),
		send:         make(chan *eventData, 100),
	}

	s.connect <- client

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
		case <-s.stop: // サーバーが停止
			close(client.send)
			break StreamFor

		case <-cn: // クライアントが切断
			close(client.send)
			s.disconnect <- client
			break StreamFor

		case message := <-client.send: // イベントを送信
			data, _ := json.Marshal(message.Payload)
			rw.Write([]byte("event: "))
			rw.Write([]byte(message.EventType))
			rw.Write([]byte("\n"))
			rw.Write([]byte("data: "))
			rw.Write(data)
			rw.Write(sseSeparator)
			fl.Flush()

		case <-t.C: // タイムアウト対策で10秒おきにコメント行を送信する
			rw.Write([]byte(":"))
			rw.Write(sseSeparator)
			fl.Flush()
		}
	}

	return nil
}
