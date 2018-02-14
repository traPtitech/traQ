package notification

import (
	"encoding/json"
	"github.com/labstack/echo"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/notification/events"
	"net/http"
	"sync"
)

var (
	sseSeparator = []byte("\n\n")
)

type sseClient struct {
	userId       uuid.UUID
	connectionId uuid.UUID
	send         chan *events.EventData
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
		v, _ := i.(map[uuid.UUID]*sseClient)
		return v, true
	} else {
		return nil, false
	}
}

func (s *clientsSyncMap) Store(key uuid.UUID, value map[uuid.UUID]*sseClient) {
	s.m.Store(key, value)
}

func (s *clientsSyncMap) Range(f func(key uuid.UUID, value map[uuid.UUID]*sseClient) bool) {
	s.m.Range(func(k, v interface{}) bool {
		key, _ := k.(uuid.UUID)
		value, _ := v.(map[uuid.UUID]*sseClient)
		return f(key, value)
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
			arr, exists := s.clients.Load(c.userId)
			if !exists {
				arr = make(map[uuid.UUID]*sseClient)
				s.clients.Store(c.userId, arr)
			}
			arr[c.connectionId] = c

		case c := <-s.disconnect:
			arr, _ := s.clients.Load(c.userId)
			delete(arr, c.connectionId)
		}
	}
}

func Stream(userId uuid.UUID, res *echo.Response) {
	client := &sseClient{
		userId:       userId,
		connectionId: uuid.NewV4(),
		send:         make(chan *events.EventData, 50),
		stop:         make(chan struct{}),
	}
	rw := res.Writer
	fl := res.Writer.(http.Flusher)
	cn := res.CloseNotify()

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
			rw.Write([]byte("event: " + message.EventType + "\n"))
			rw.Write([]byte("data: "))
			rw.Write(data)
			rw.Write(sseSeparator)
			fl.Flush()
		}
	}
}
