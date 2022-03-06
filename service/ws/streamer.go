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
	"github.com/traPtitech/traQ/service/viewer"
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
	hub      *hub.Hub
	vm       *viewer.Manager
	webrtc   *webrtcv3.Manager
	logger   *zap.Logger
	sessions map[*session]struct{}
	closed   bool
	mu       sync.RWMutex
}

// NewStreamer WebSocketストリーマーを生成し起動します
func NewStreamer(hub *hub.Hub, vm *viewer.Manager, webrtc *webrtcv3.Manager, logger *zap.Logger) *Streamer {
	h := &Streamer{
		hub:      hub,
		vm:       vm,
		webrtc:   webrtc,
		logger:   logger.Named("ws"),
		sessions: make(map[*session]struct{}),
		closed:   false,
	}
	return h
}

func (s *Streamer) register(session *session) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[session] = struct{}{}
}

func (s *Streamer) unregister(session *session) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, session)
}

// IterateSessions 全セッションをイテレートします
func (s *Streamer) IterateSessions(f func(session Session)) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for session := range s.sessions {
		f(session)
	}
}

// WriteMessage 指定したセッションにメッセージを書き込みます
func (s *Streamer) WriteMessage(t string, body interface{}, targetFunc TargetFunc) {
	m := &rawMessage{
		t:    websocket.TextMessage,
		data: makeMessage(t, body).toJSON(),
	}
	s.mu.RLock()
	for session := range s.sessions {
		if targetFunc(session) {
			if err := session.writeMessage(m); err != nil {
				if err == ErrBufferIsFull {
					s.logger.Warn("Discard a message because the session's buffer is full.",
						zap.String("type", t), zap.Any("body", body),
						zap.Stringer("userID", session.userID))
					continue
				}
			}
		}
	}
	s.mu.RUnlock()
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

	session := &session{
		key:      random.AlphaNumeric(20),
		req:      r,
		conn:     conn,
		closed:   false,
		streamer: s,
		send:     make(chan *rawMessage, messageBufferSize),
		userID:   r.Context().Value(ctxkey.UserID).(uuid.UUID),
	}

	s.register(session)
	s.hub.Publish(hub.Message{
		Name: event.WSConnected,
		Fields: hub.Fields{
			"user_id": session.UserID(),
			"req":     r,
		},
	})

	go session.writeLoop()
	session.readLoop()

	s.vm.RemoveViewer(session)
	_ = s.webrtc.ResetState(session.Key(), session.UserID())
	s.hub.Publish(hub.Message{
		Name: event.WSDisconnected,
		Fields: hub.Fields{
			"user_id": session.UserID(),
			"req":     r,
		},
	})
	s.unregister(session)
	session.close()
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
	for session := range s.sessions {
		_ = session.writeMessage(m)
		session.close()
	}
	s.sessions = make(map[*session]struct{})
	return nil
}
