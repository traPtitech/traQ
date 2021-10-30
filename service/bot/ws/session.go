package ws

import (
	"net/http"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
)

type session struct {
	key    string
	userID uuid.UUID

	sync.RWMutex

	req      *http.Request
	conn     *websocket.Conn
	open     bool
	streamer *Streamer
	send     chan *rawMessage
}

func (s *session) readLoop() {
	s.conn.SetReadLimit(maxReadMessageSize)
	_ = s.conn.SetReadDeadline(time.Now().Add(pongWait))
	s.conn.SetPongHandler(func(string) error {
		_ = s.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		t, m, err := s.conn.ReadMessage()
		if err != nil {
			break
		}

		if t == websocket.TextMessage {
			s.commandHandler(string(m))
		}

		if t == websocket.BinaryMessage {
			// unsupported
			_ = s.writeMessage(&rawMessage{t: websocket.CloseMessage, data: websocket.FormatCloseMessage(websocket.CloseUnsupportedData, "binary message is not supported.")})
			break
		}
	}
}

func (s *session) writeLoop() {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case msg, ok := <-s.send:
			if !ok {
				return
			}

			if err := s.write(msg.t, msg.data); err != nil {
				return
			}

			if msg.t == websocket.CloseMessage {
				return
			}

		case <-ticker.C:
			_ = s.write(websocket.PingMessage, []byte{})
		}
	}
}

func (s *session) writeMessage(msg *rawMessage) (err error) {
	s.RLock()
	defer s.RUnlock()
	if !s.open {
		return ErrAlreadyClosed
	}

	select {
	case s.send <- msg:
	default:
		return ErrBufferIsFull
	}
	return nil
}

func (s *session) write(messageType int, data []byte) error {
	_ = s.conn.SetWriteDeadline(time.Now().Add(writeWait))
	return s.conn.WriteMessage(messageType, data)
}

func (s *session) close() {
	s.Lock()
	defer s.Unlock()
	if s.open {
		s.open = false
		s.conn.Close()
		close(s.send)
	}
}
