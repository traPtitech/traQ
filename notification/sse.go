package notification

import (
	"encoding/json"
	"github.com/labstack/echo"
	"github.com/satori/go.uuid"
	"net/http"
	"sync"
	"time"
)

var (
	sseSeparator = []byte("\n\n")
)

type sseClient struct {
	userID       uuid.UUID
	connectionID uuid.UUID
	send         chan *eventData
	stop         chan struct{}
}

type sseStreamer struct {
	clients    *clientsSyncMap
	newConnect chan *sseClient
	disconnect chan *sseClient
	stop       chan struct{}
}

type clientsSyncMap struct {
	m sync.Map
}

func (s *clientsSyncMap) Load(key uuid.UUID) (map[uuid.UUID]*sseClient, bool) {
	i, ok := s.m.Load(key)
	if ok {
		return i.(map[uuid.UUID]*sseClient), true
	}
	return nil, false
}

func (s *clientsSyncMap) Store(key uuid.UUID, value map[uuid.UUID]*sseClient) {
	s.m.Store(key, value)
}

func (s *clientsSyncMap) Range(f func(key uuid.UUID, value map[uuid.UUID]*sseClient) bool) {
	s.m.Range(func(k, v interface{}) bool {
		return f(k.(uuid.UUID), v.(map[uuid.UUID]*sseClient))
	})
}

func newSseStreamer() *sseStreamer {
	return &sseStreamer{
		clients:    &clientsSyncMap{},
		newConnect: make(chan *sseClient),
		disconnect: make(chan *sseClient, 10),
		stop:       make(chan struct{}),
	}
}

func (s *sseStreamer) run() {
	for {
		select {
		case <-s.stop:
			close(s.newConnect)
			close(s.disconnect)
			s.clients = nil
			return

		case c := <-s.newConnect:
			arr, exists := s.clients.Load(c.userID)
			if !exists {
				arr = make(map[uuid.UUID]*sseClient)
				s.clients.Store(c.userID, arr)
			}
			arr[c.connectionID] = c

		case c := <-s.disconnect:
			arr, _ := s.clients.Load(c.userID)
			delete(arr, c.connectionID)
		}
	}
}

//Stream 指定したユーザーIDへのイベントを*echo.Responseに流します。
func Stream(userID uuid.UUID, res *echo.Response) {
	client := &sseClient{
		userID:       userID,
		connectionID: uuid.NewV4(),
		send:         make(chan *eventData, 50),
		stop:         make(chan struct{}),
	}
	rw := res.Writer
	fl := res.Writer.(http.Flusher)
	cn := res.CloseNotify()
	mu := sync.Mutex{}

	select {
	case <-streamer.stop:
		rw.Write([]byte("event: CONNECTION_FAILED"))
		rw.Write(sseSeparator)
		fl.Flush()
		return

	default:
		streamer.newConnect <- client
		rw.Write([]byte("event: CONNECTED"))
		rw.Write(sseSeparator)
		fl.Flush()
	}

	// proxyに切られる問題の対策
	go func() {
		defer func() {
			recover()
		}()

		for {
			time.Sleep(10 * time.Second)
			select {
			case <-streamer.stop:
				return
			case <-cn:
				return
			default:
				mu.Lock()
				rw.Write([]byte(":"))
				rw.Write(sseSeparator)
				fl.Flush()
				mu.Unlock()
			}
		}
	}()

	for {
		select {
		case <-streamer.stop:
			close(client.stop)
			close(client.send)
			return

		case <-cn:
			close(client.stop)
			close(client.send)
			streamer.disconnect <- client
			return

		case message := <-client.send:
			//message.payload is not unsupported type or unsupported value.
			data, _ := json.Marshal(message.Payload)
			mu.Lock()
			rw.Write([]byte("event: "))
			rw.Write([]byte(message.EventType))
			rw.Write([]byte("\n"))
			rw.Write([]byte("data: "))
			rw.Write(data)
			rw.Write(sseSeparator)
			fl.Flush()
			mu.Unlock()
		}
	}
}
