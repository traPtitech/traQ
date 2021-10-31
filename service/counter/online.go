package counter

import (
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/traPtitech/traQ/event"
)

var (
	onlineUsersCounter = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "traq",
		Name:      "online_users",
	}, []string{"user_type"})
	wsConnectionCounter = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "traq",
		Name:      "ws_connections",
	}, []string{"user_type"})
)

// OnlineCounter オンラインユーザーカウンター
type OnlineCounter struct {
	hub          *hub.Hub
	counters     map[uuid.UUID]*counter
	countersLock sync.Mutex
}

// NewOnlineCounter オンラインユーザーカウンターを生成します
func NewOnlineCounter(hub *hub.Hub) *OnlineCounter {
	oc := &OnlineCounter{
		hub:      hub,
		counters: map[uuid.UUID]*counter{},
	}
	go func() {
		for e := range hub.Subscribe(8, event.WSConnected, event.WSDisconnected, event.BotWSConnected, event.BotWSDisconnected).Receiver {
			switch e.Topic() {
			case event.WSConnected:
				oc.inc(e.Fields["user_id"].(uuid.UUID), "user")
				wsConnectionCounter.WithLabelValues("user").Inc()
			case event.BotWSConnected:
				oc.inc(e.Fields["user_id"].(uuid.UUID), "bot")
				wsConnectionCounter.WithLabelValues("bot").Inc()
			case event.WSDisconnected:
				oc.dec(e.Fields["user_id"].(uuid.UUID), "user")
				wsConnectionCounter.WithLabelValues("user").Dec()
			case event.BotWSDisconnected:
				oc.dec(e.Fields["user_id"].(uuid.UUID), "bot")
				wsConnectionCounter.WithLabelValues("bot").Dec()
			}
		}
	}()
	return oc
}

// inc 指定したユーザーのカウンタをインクリメントします
func (oc *OnlineCounter) inc(userID uuid.UUID, userType string) (toOnline bool) {
	oc.countersLock.Lock()
	c, ok := oc.counters[userID]
	if !ok {
		c = &counter{
			userID: userID,
		}
		oc.counters[userID] = c
	}
	oc.countersLock.Unlock()

	toOnline = c.inc()
	if toOnline {
		onlineUsersCounter.WithLabelValues(userType).Inc()
		oc.hub.Publish(hub.Message{
			Name: event.UserOnline,
			Fields: hub.Fields{
				"user_id":  userID,
				"datetime": c.getLastUpdated(),
			},
		})
	}
	return
}

// dec 指定したユーザーのカウンタをデクリメントします
func (oc *OnlineCounter) dec(userID uuid.UUID, userType string) (toOffline bool) {
	oc.countersLock.Lock()
	c, ok := oc.counters[userID]
	if !ok {
		oc.countersLock.Unlock()
		return
	}
	oc.countersLock.Unlock()

	toOffline = c.dec()
	if toOffline {
		onlineUsersCounter.WithLabelValues(userType).Dec()
		oc.hub.Publish(hub.Message{
			Name: event.UserOffline,
			Fields: hub.Fields{
				"user_id":  userID,
				"datetime": c.getLastUpdated(),
			},
		})
	}
	return
}

// IsOnline 指定したユーザーがオンラインかどうかを取得します
func (oc *OnlineCounter) IsOnline(userID uuid.UUID) bool {
	oc.countersLock.Lock()
	c, ok := oc.counters[userID]
	oc.countersLock.Unlock()

	return ok && c.isOnline()
}

// GetOnlineUserIDs オンラインなユーザーのUUIDの配列を取得します
func (oc *OnlineCounter) GetOnlineUserIDs() []uuid.UUID {
	oc.countersLock.Lock()
	users := make([]uuid.UUID, 0, len(oc.counters))
	for u, c := range oc.counters {
		if c.isOnline() {
			users = append(users, u)
		}
	}
	oc.countersLock.Unlock()
	return users
}

type counter struct {
	sync.RWMutex
	userID      uuid.UUID
	count       int
	lastUpdated time.Time
}

func (s *counter) isOnline() (r bool) {
	s.RLock()
	r = s.count > 0
	s.RUnlock()
	return
}

func (s *counter) inc() (toOnline bool) {
	s.Lock()
	s.count++
	s.lastUpdated = time.Now()
	if s.count == 1 {
		toOnline = true
	}
	s.Unlock()
	return
}

func (s *counter) dec() (toOffline bool) {
	s.Lock()
	if s.count > 0 {
		s.count--
		s.lastUpdated = time.Now()
		if s.count == 0 {
			toOffline = true
		}
	}
	s.Unlock()
	return
}

func (s *counter) getLastUpdated() (t time.Time) {
	s.RLock()
	t = s.lastUpdated
	s.RUnlock()
	return
}
