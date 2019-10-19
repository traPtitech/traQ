package realtime

import (
	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/traPtitech/traQ/event"
	"sync"
	"time"
)

var onlineUsersCounter = promauto.NewGauge(prometheus.GaugeOpts{
	Namespace: "traq",
	Name:      "online_users",
})

// OnlineCounter オンラインユーザーカウンター
type OnlineCounter struct {
	hub          *hub.Hub
	counters     map[uuid.UUID]*counter
	countersLock sync.Mutex
}

func newOnlineCounter(hub *hub.Hub) *OnlineCounter {
	oc := &OnlineCounter{
		hub:      hub,
		counters: map[uuid.UUID]*counter{},
	}
	return oc
}

// Inc 指定したユーザーのカウンタをインクリメントします
func (oc *OnlineCounter) Inc(userID uuid.UUID) (toOnline bool) {
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
		onlineUsersCounter.Inc()
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

// Dec 指定したユーザーのカウンタをデクリメントします
func (oc *OnlineCounter) Dec(userID uuid.UUID) (toOffline bool) {
	oc.countersLock.Lock()
	c, ok := oc.counters[userID]
	if !ok {
		oc.countersLock.Unlock()
		return
	}
	oc.countersLock.Unlock()

	toOffline = c.dec()
	if toOffline {
		onlineUsersCounter.Dec()
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
	if !ok {
		oc.countersLock.Unlock()
		return false
	}
	oc.countersLock.Unlock()

	return c.isOnline()
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
