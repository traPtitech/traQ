package ws

import (
	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
	"github.com/traPtitech/traQ/service/viewer"
	"net/http"
	"sync"
	"time"
)

// Session WebSocketセッション
type Session interface {
	Key() string
	// UserID このセッションのUserID
	UserID() uuid.UUID
	// State このセッションのチャンネル閲覧状態
	ViewState() (channelID uuid.UUID, state viewer.State)
	// TimelineStreaming このセッションのタイムラインストリーミングが有効かどうか
	TimelineStreaming() bool
}

type session struct {
	key    string
	userID uuid.UUID

	viewState struct {
		channelID uuid.UUID
		state     viewer.State
	}
	enabledTimelineStreaming bool
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

func (s *session) writeMessage(msg *rawMessage) error {
	if s.closed() {
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
	if !s.closed() {
		s.Lock()
		s.open = false
		s.conn.Close()
		close(s.send)
		s.Unlock()
	}
}

func (s *session) closed() bool {
	s.RLock()
	defer s.RUnlock()

	return !s.open
}

// Key implements Session interface.
func (s *session) Key() string {
	return s.key
}

// UserID implements Session interface.
func (s *session) UserID() uuid.UUID {
	return s.userID
}

// ViewState implements Session interface.
func (s *session) ViewState() (uuid.UUID, viewer.State) {
	s.RLock()
	defer s.RUnlock()
	return s.viewState.channelID, s.viewState.state
}

// TimelineStreaming implements Session interface.
func (s *session) TimelineStreaming() bool {
	s.RLock()
	defer s.RUnlock()
	return s.enabledTimelineStreaming
}

func (s *session) setViewState(cid uuid.UUID, state viewer.State) {
	s.Lock()
	defer s.Unlock()
	s.viewState.channelID = cid
	s.viewState.state = state
}

func (s *session) setTimelineStreaming(enabled bool) {
	s.Lock()
	defer s.Unlock()
	s.enabledTimelineStreaming = enabled
}
