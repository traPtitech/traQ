package ws

import (
	"errors"
	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
	"github.com/leandro-lugaresi/hub"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/router/extension"
	"github.com/traPtitech/traQ/service/viewer"
	"github.com/traPtitech/traQ/service/webrtcv3"
	"github.com/traPtitech/traQ/utils/random"
	"go.uber.org/zap"
	"net/http"
	"sync"
)

var (
	// ErrAlreadyClosed 既に閉じられています
	ErrAlreadyClosed = errors.New("already closed")
	// ErrBufferIsFull 送信バッファが溢れました
	ErrBufferIsFull = errors.New("buffer is full")

	wsConnectionCounter = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "traq",
		Name:      "ws_connections",
	})
)

// Streamer WebSocketストリーマー
type Streamer struct {
	hub        *hub.Hub
	vm         *viewer.Manager
	webrtc     *webrtcv3.Manager
	logger     *zap.Logger
	sessions   map[*session]struct{}
	register   chan *session
	unregister chan *session
	stop       chan struct{}
	open       bool
	mu         sync.RWMutex
}

// NewStreamer WebSocketストリーマーを生成し起動します
func NewStreamer(hub *hub.Hub, vm *viewer.Manager, webrtc *webrtcv3.Manager, logger *zap.Logger) *Streamer {
	h := &Streamer{
		hub:        hub,
		vm:         vm,
		webrtc:     webrtc,
		logger:     logger.Named("ws"),
		sessions:   make(map[*session]struct{}),
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
			s.sessions[session] = struct{}{}
			s.mu.Unlock()

		case session := <-s.unregister:
			if _, ok := s.sessions[session]; ok {
				s.mu.Lock()
				delete(s.sessions, session)
				s.mu.Unlock()
			}

		case <-s.stop:
			s.mu.Lock()
			m := &rawMessage{
				t:    websocket.CloseMessage,
				data: websocket.FormatCloseMessage(websocket.CloseServiceRestart, "Server is stopping..."),
			}
			for session := range s.sessions {
				_ = session.writeMessage(m)
				delete(s.sessions, session)
				session.close()
			}
			s.open = false
			s.mu.Unlock()
			return
		}
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
		userID:   r.Context().Value(extension.CtxUserIDKey).(uuid.UUID),
	}

	s.register <- session
	wsConnectionCounter.Inc()
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
	wsConnectionCounter.Dec()
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
