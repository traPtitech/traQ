package ws

import (
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"

	"github.com/traPtitech/traQ/service/viewer"
	"github.com/traPtitech/traQ/utils/random"
)

// Session WebSocketセッション
type Session interface {
	Key() string
	// UserID このセッションのUserID
	UserID() uuid.UUID
	// ViewState このセッションのチャンネル閲覧状態
	ViewState() (channelID uuid.UUID, state viewer.State)
	// TimelineStreaming このセッションのタイムラインストリーミングが有効かどうか
	TimelineStreaming() bool
}

type session struct {
	key      string
	userID   uuid.UUID
	conn     *websocket.Conn
	streamer *Streamer

	viewState struct {
		channelID uuid.UUID
		state     viewer.State
	}
	enabledTimelineStreaming bool

	*sync.RWMutex
	send      chan *rawMessage
	closed    bool
	closeWait *sync.Cond
}

func newSession(userID uuid.UUID, conn *websocket.Conn, streamer *Streamer) *session {
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

			if msg.t == websocket.CloseMessage {
				return
			}

		case <-ticker.C:
			_ = s.write(websocket.PingMessage, []byte{})
		}
	}
}

func (s *session) WriteMessage(msg *rawMessage) error {
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
