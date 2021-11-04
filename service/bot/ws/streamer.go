package ws

import (
	"errors"
	"net/http"
	"sync"

	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
	"github.com/leandro-lugaresi/hub"
	"go.uber.org/zap"

	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/router/extension/ctxkey"
	"github.com/traPtitech/traQ/service/webrtcv3"
	"github.com/traPtitech/traQ/utils/random"
)

var (
	// ErrAlreadyClosed 既に閉じられています
	ErrAlreadyClosed = errors.New("already closed")
	// ErrBufferIsFull 送信バッファが溢れました
	ErrBufferIsFull = errors.New("buffer is full")
)

// Streamer WebSocketストリーマー
type Streamer struct {
	hub        *hub.Hub
	webrtc     *webrtcv3.Manager
	logger     *zap.Logger
	sessions   map[uuid.UUID][]*session
	register   chan *session
	unregister chan *session
	stop       chan struct{}
	open       bool
	mu         sync.RWMutex
}

// NewStreamer WebSocketストリーマーを生成し起動します
func NewStreamer(hub *hub.Hub, webrtc *webrtcv3.Manager, logger *zap.Logger) *Streamer {
	h := &Streamer{
		hub:        hub,
		webrtc:     webrtc,
		logger:     logger.Named("bot.ws"),
		sessions:   make(map[uuid.UUID][]*session),
		register:   make(chan *session),
		unregister: make(chan *session),
		stop:       make(chan struct{}),
		open:       true,
	}

	go h.run()
	return h
}

func (s *Streamer) run() {
	for {
		select {
		case session := <-s.register:
			s.mu.Lock()
			s.sessions[session.userID] = append(s.sessions[session.userID], session)
			s.mu.Unlock()

		case session := <-s.unregister:
			s.mu.Lock()
			if sessions, ok := s.sessions[session.userID]; ok {
				s.sessions[session.userID] = filterSession(sessions, session)
				if len(s.sessions[session.userID]) == 0 {
					delete(s.sessions, session.userID)
				}
			}
			s.mu.Unlock()

		case <-s.stop:
			s.mu.Lock()
			m := &rawMessage{
				t:    websocket.CloseMessage,
				data: websocket.FormatCloseMessage(websocket.CloseServiceRestart, "Server is stopping..."),
			}
			for _, sessions := range s.sessions {
				for _, session := range sessions {
					_ = session.writeMessage(m)
					session.close()
				}
			}
			s.sessions = make(map[uuid.UUID][]*session)
			s.open = false
			s.mu.Unlock()
			return
		}
	}
}

func filterSession(sessions []*session, target *session) []*session {
	s := make([]*session, 0, len(sessions)-1)
	for _, session := range sessions {
		if session != target {
			s = append(s, session)
		}
	}
	return s
}

// WriteMessage 指定したセッションにメッセージを書き込みます
func (s *Streamer) WriteMessage(t string, reqID uuid.UUID, body []byte, botUserID uuid.UUID) (errs []error, attempted bool) {
	m := &rawMessage{
		t:    websocket.TextMessage,
		data: makeEventMessage(t, reqID, body).toJSON(),
	}
	s.mu.RLock()
	for _, session := range s.sessions[botUserID] {
		if err := session.writeMessage(m); err != nil {
			errs = append(errs, err)
			if err == ErrBufferIsFull {
				s.logger.Warn("Discarded a message because the session's buffer was full.",
					zap.String("type", t),
					zap.Stringer("reqID", reqID),
					zap.Any("body", body),
					zap.Stringer("userID", session.userID))
			}
		}
		attempted = true
	}
	s.mu.RUnlock()
	return
}

// ServeHTTP http.Handlerインターフェイスの実装
func (s *Streamer) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	if s.IsClosed() {
		http.Error(rw, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		return
	}

	conn, err := upgrader.Upgrade(rw, r, rw.Header())
	if err != nil {
		return
	}

	session := &session{
		key:      random.AlphaNumeric(20),
		req:      r,
		conn:     conn,
		open:     true,
		streamer: s,
		send:     make(chan *rawMessage, messageBufferSize),
		userID:   r.Context().Value(ctxkey.UserID).(uuid.UUID),
	}

	s.register <- session
	s.hub.Publish(hub.Message{
		Name: event.BotWSConnected,
		Fields: hub.Fields{
			"user_id": session.userID,
			"req":     r,
		},
	})

	go session.writeLoop()
	session.readLoop()

	_ = s.webrtc.ResetState(session.key, session.userID)
	s.hub.Publish(hub.Message{
		Name: event.BotWSDisconnected,
		Fields: hub.Fields{
			"user_id": session.userID,
			"req":     r,
		},
	})
	s.unregister <- session
	session.close()
}

// IsClosed ストリーマーが停止しているかどうか
func (s *Streamer) IsClosed() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return !s.open
}

// Close ストリーマーを停止します
func (s *Streamer) Close() error {
	if s.IsClosed() {
		return ErrAlreadyClosed
	}
	s.stop <- struct{}{}
	return nil
}
