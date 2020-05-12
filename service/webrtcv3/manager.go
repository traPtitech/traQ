package webrtcv3

import (
	"errors"
	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/traPtitech/traQ/event"
	"sync"
)

var (
	ErrOccupied             = errors.New("connection has already existed")
	webrtcUsingUsersCounter = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "traq",
		Name:      "webrtc_using_users",
	})
	webrtcUsingChannelsCounter = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "traq",
		Name:      "webrtc_using_channels",
	})
)

// Manager WebRTCマネージャー
type Manager struct {
	eventbus      *hub.Hub
	userStates    map[uuid.UUID]*userState
	channelStates map[uuid.UUID]*channelState
	statesLock    sync.RWMutex
}

// NewManager WebRTCマネージャーを生成します
func NewManager(eventbus *hub.Hub) *Manager {
	manager := &Manager{
		eventbus:      eventbus,
		userStates:    map[uuid.UUID]*userState{},
		channelStates: map[uuid.UUID]*channelState{},
	}
	return manager
}

// IterateStates 全状態をイテレートします
func (m *Manager) IterateStates(f func(state ChannelState)) {
	m.statesLock.RLock()
	defer m.statesLock.RUnlock()
	for _, state := range m.channelStates {
		f(state)
	}
}

// SetState 指定した状態をセットします
func (m *Manager) SetState(connKey string, user, channel uuid.UUID, sessions map[string]string) error {
	if len(sessions) == 0 {
		return m.ResetState(connKey, user)
	}

	m.statesLock.Lock()
	defer m.statesLock.Unlock()

	us, ok := m.userStates[user]
	if !ok {
		us = &userState{
			connKey: connKey,
			userID:  user,
		}
		m.userStates[user] = us
		webrtcUsingUsersCounter.Inc()
	}

	if us.valid() && us.channelID != channel {
		m.channelStates[us.channelID].removeUser(user)
	}

	cs, ok := m.channelStates[channel]
	if !ok {
		cs = &channelState{
			channelID: channel,
			users:     map[uuid.UUID]*userState{},
		}
		m.channelStates[channel] = cs
		webrtcUsingChannelsCounter.Inc()
	}

	us.sessions = sessions
	us.channelID = channel
	cs.setUser(us)

	m.eventbus.Publish(hub.Message{
		Name: event.UserWebRTCv3StateChanged,
		Fields: hub.Fields{
			"user_id":    us.userID,
			"channel_id": us.channelID,
			"sessions":   us.sessions,
		},
	})
	return nil
}

// ResetState 指定したユーザーの状態を削除します
func (m *Manager) ResetState(connKey string, user uuid.UUID) error {
	m.statesLock.Lock()
	defer m.statesLock.Unlock()

	us, ok := m.userStates[user]
	if !ok {
		return nil
	}

	if us.connKey != connKey {
		return ErrOccupied
	}

	delete(m.userStates, user)
	webrtcUsingUsersCounter.Dec()
	cs := m.channelStates[us.channelID]
	cs.removeUser(user)
	if !cs.valid() {
		delete(m.channelStates, cs.channelID)
		webrtcUsingChannelsCounter.Dec()
	}

	m.eventbus.Publish(hub.Message{
		Name: event.UserWebRTCv3StateChanged,
		Fields: hub.Fields{
			"user_id":    us.userID,
			"channel_id": us.channelID,
			"sessions":   map[string]string{},
		},
	})
	return nil
}
