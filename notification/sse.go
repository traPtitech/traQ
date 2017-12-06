package notification

import (
	"errors"
	"fmt"
	"github.com/labstack/echo"
	"github.com/satori/go.uuid"
	"net/http"
)

type sseMessage []byte

type sseClient struct {
	userId       uuid.UUID
	connectionId uuid.UUID
	send         chan []byte
}

type sseStreamer struct {
	isRunning  bool
	clients    map[uuid.UUID]map[uuid.UUID]*sseClient
	newConnect chan *sseClient
	disconnect chan *sseClient
	event      chan *sseMessage
	stop       chan struct{}
}

func makeSSEStreamer() (s *sseStreamer) {
	s = &sseStreamer{
		isRunning:  false,
		clients:    make(map[uuid.UUID]map[uuid.UUID]*sseClient, 200),
		newConnect: make(chan *sseClient),
		disconnect: make(chan *sseClient, 10),
		event:      make(chan *sseMessage, 100),
		stop:       make(chan struct{}),
	}
	return
}

func (s *sseStreamer) run() error {
	if s.isRunning {
		return errors.New("already running")
	}

	s.isRunning = true
	defer func() {
		s.isRunning = false
	}()

Loop:
	for {
		select {
		case <-s.stop:
			break Loop

		case c := <-s.newConnect:
			arr, exists := s.clients[c.userId]
			if !exists {
				arr = make(map[uuid.UUID]*sseClient)
				s.clients[c.userId] = arr
			}
			arr[c.connectionId] = c
			fmt.Println("connect")

		case c := <-s.disconnect:
			arr := s.clients[c.userId]
			delete(arr, c.connectionId)
			fmt.Println("disconnect")

			//case event := <-s.event:

		}
	}

	return nil
}

func (m *sseMessage) send(clients map[uuid.UUID]*sseClient) {

}

func (s *sseStreamer) Stream(userId uuid.UUID, res *echo.Response) {
	client := &sseClient{
		userId:       userId,
		connectionId: uuid.NewV4(),
		send:         make(chan []byte),
	}
	rw := res.Writer
	fl := res.Writer.(http.Flusher)
	cn := res.CloseNotify()

	s.newConnect <- client
	for {
		select {
		case <-cn:
			s.disconnect <- client
			return

		case message := <-client.send:
			rw.Write(message)
			fl.Flush()
		}
	}
}