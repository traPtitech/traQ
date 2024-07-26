package ws

import (
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"

	"github.com/traPtitech/traQ/utils/random"
)

type session struct {
	key      string
	userID   uuid.UUID
	conn     *websocket.Conn
	streamer *Streamer

	*sync.RWMutex
	send      chan *rawMessage
	closed    bool
	closeWait *sync.Cond
}

func newSession(userID uuid.UUID, streamer *Streamer, conn *websocket.Conn) *session {
	mu := sync.RWMutex{}
	return &session{
		key:      random.AlphaNumeric(20),
		userID:   userID,
		conn:     conn,
		streamer: streamer,

		RWMutex:   &mu,
		send:      make(chan *rawMessage, messageBufferSize),
		closed:    false,
		closeWait: sync.NewCond(&mu),
	}
}

func (s *session) ReadLoop() {
	defer s.close()

	s.conn.SetReadLimit(maxReadMessageSize)
	_ = s.conn.SetReadDeadline(time.Now().Add(pongWait))
	s.conn.SetPongHandler(func(string) error {
		_ = s.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		t, m, err := s.conn.ReadMessage()
		if err != nil {
			return
		}
		incWebSocketReadBytesTotal(s.userID, len(m))

		if t == websocket.TextMessage {
			s.commandHandler(string(m))
		}

		if t == websocket.BinaryMessage {
			// unsupported
			_ = s.WriteMessage(&rawMessage{t: websocket.CloseMessage, data: websocket.FormatCloseMessage(websocket.CloseUnsupportedData, "binary message is not supported.")})
			return
		}
	}
}

func (s *session) WriteLoop() {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	defer s.close()

	for {
		select {
		case msg, ok := <-s.send:
			if !ok {
				return
			}

			if err := s.write(msg.t, msg.data); err != nil {
				return
			}
			incWebSocketWriteBytesTotal(s.userID, len(msg.data))

			if msg.t == websocket.CloseMessage {
				return
			}

		case <-ticker.C:
			if err := s.write(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

func (s *session) WriteMessage(msg *rawMessage) (err error) {
	s.RLock()
	defer s.RUnlock()
	if s.closed {
		return ErrAlreadyClosed
	}

	select {
	case s.send <- msg:
		return nil
	default:
		return ErrBufferIsFull
	}
}

func (s *session) write(messageType int, data []byte) error {
	_ = s.conn.SetWriteDeadline(time.Now().Add(writeWait))
	return s.conn.WriteMessage(messageType, data)
}

func (s *session) close() {
	s.Lock()
	defer s.Unlock()

	if !s.closed {
		s.closed = true
		s.closeWait.Broadcast()
		_ = s.conn.Close()
		close(s.send)
	}
}

func (s *session) WaitForClose() {
	s.Lock()
	for !s.closed {
		s.closeWait.Wait()
	}
	s.Unlock()
}
