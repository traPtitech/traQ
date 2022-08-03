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
	"github.com/traPtitech/traQ/router/extension/ctxKey"
	"github.com/traPtitech/traQ/service/webrtcv3"
)

var (
	// ErrAlreadyClosed 既に閉じられています
	ErrAlreadyClosed = errors.New("already closed")
	// ErrBufferIsFull 送信バッファが溢れました
	ErrBufferIsFull = errors.New("buffer is full")
)

// Streamer WebSocketストリーマー
type Streamer struct {
	hub      *hub.Hub
	webrtc   *webrtcv3.Manager
	logger   *zap.Logger
	sessions map[uuid.UUID][]*session
	closed   bool
	mu       sync.RWMutex
}

// NewStreamer WebSocketストリーマーを生成し起動します
func NewStreamer(hub *hub.Hub, webrtc *webrtcv3.Manager, logger *zap.Logger) *Streamer {
	h := &Streamer{
		hub:      hub,
		webrtc:   webrtc,
		logger:   logger.Named("bot.ws"),
		sessions: make(map[uuid.UUID][]*session),
		closed:   false,
	}
	return h
}

func (s *Streamer) register(session *session) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[session.userID] = append(s.sessions[session.userID], session)
}

func (s *Streamer) unregister(session *session) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sessions, ok := s.sessions[session.userID]; ok {
		s.sessions[session.userID] = filterSession(sessions, session)
		if len(s.sessions[session.userID]) == 0 {
			delete(s.sessions, session.userID)
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
		if err := session.WriteMessage(m); err != nil {
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
	s.mu.RLock()
	if s.closed {
		http.Error(rw, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		s.mu.RUnlock()
		return
	}
	s.mu.RUnlock()

	conn, err := upgrader.Upgrade(rw, r, rw.Header())
	if err != nil {
		return
	}

	session := newSession(r.Context().Value(ctxKey.UserID).(uuid.UUID), s, conn)

	s.register(session)
	s.hub.Publish(hub.Message{
		Name: event.BotWSConnected,
		Fields: hub.Fields{
			"user_id": session.userID,
			"req":     r,
		},
	})

	go session.WriteLoop()
	session.ReadLoop()

	_ = s.webrtc.ResetState(session.key, session.userID)
	s.hub.Publish(hub.Message{
		Name: event.BotWSDisconnected,
		Fields: hub.Fields{
			"user_id": session.userID,
			"req":     r,
		},
	})
	s.unregister(session)
}

// Close ストリーマーを停止します
func (s *Streamer) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return ErrAlreadyClosed
	}
	s.closed = true

	m := &rawMessage{
		t:    websocket.CloseMessage,
		data: websocket.FormatCloseMessage(websocket.CloseServiceRestart, "Server is stopping..."),
	}

	var wg sync.WaitGroup
	for _, sessions := range s.sessions {
		for _, s := range sessions {
			wg.Add(1)
			go func(s *session) {
				defer wg.Done()
				if err := s.WriteMessage(m); err != nil {
					return
				}
				s.WaitForClose()
			}(s)
		}
	}
	wg.Wait()

	s.sessions = make(map[uuid.UUID][]*session)
	return nil
}
