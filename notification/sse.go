package notification

import (
	"encoding/json"
	"github.com/labstack/echo"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/notification/events"
	"net/http"
)

type sseClient struct {
	userId       uuid.UUID
	connectionId uuid.UUID
	send         chan *events.EventData
	close        chan struct{}
}

type sseStreamer struct {
	clients    map[userId]map[uuid.UUID]*sseClient
	newConnect chan *sseClient
	disconnect chan *sseClient
	stop       chan struct{}
}

func (s *sseStreamer) run() {
	for {
		select {
		case <-s.stop:
			for _, u := range s.clients {
				for _, c := range u {
					c.close <- struct{}{}
				}
			}
			return

		case c := <-s.newConnect:
			arr, exists := s.clients[c.userId]
			if !exists {
				arr = make(map[uuid.UUID]*sseClient)
				s.clients[c.userId] = arr
			}
			arr[c.connectionId] = c

		case c := <-s.disconnect:
			arr := s.clients[c.userId]
			delete(arr, c.connectionId)

		}
	}
}

func GetNotificationStream(c echo.Context) error {
	userId := uuid.FromStringOrNil(c.Get("user").(*model.User).ID)

	if _, ok := c.Response().Writer.(http.Flusher); !ok {
		return echo.NewHTTPError(http.StatusNotImplemented, "Server Sent Events is not supported.")
	}

	//Set headers for SSE
	c.Response().Header().Set(echo.HeaderContentType, "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")
	c.Response().WriteHeader(http.StatusOK)

	Stream(userId, c.Response())
	return nil
}

func Stream(userId uuid.UUID, res *echo.Response) {
	client := &sseClient{
		userId:       userId,
		connectionId: uuid.NewV4(),
		send:         make(chan *events.EventData, 10),
		close:        make(chan struct{}),
	}
	rw := res.Writer
	fl := res.Writer.(http.Flusher)
	cn := res.CloseNotify()

	rw.Write([]byte("event: CONNECTED\n\n"))
	fl.Flush()

	streamer.newConnect <- client
	for {
		select {
		case <-cn:
			streamer.disconnect <- client
			return

		case <-client.close:
			streamer.disconnect <- client
			return

		case message := <-client.send:
			//message.payload is not unsupported type or unsupported value.
			data, _ := json.Marshal(message.Payload)
			rw.Write([]byte("event: " + message.EventType + "\n"))
			rw.Write([]byte("data: "))
			rw.Write(data)
			rw.Write([]byte("\n\n"))
			fl.Flush()
		}
	}
}
