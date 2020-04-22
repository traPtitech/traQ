package webrtcv3

import (
	"errors"
	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/event"
	"sync"
)

var ErrOccupied = errors.New("connection has already existed")

// Manager WebRTCマネージャー
type Manager struct {
	eventbus      *hub.Hub
	userStates    map[uuid.UUID]*UserState
	channelStates map[uuid.UUID]*ChannelState
	statesLock    sync.RWMutex
}

// NewManager WebRTCマネージャーを生成します
func NewManager(eventbus *hub.Hub) *Manager {
	manager := &Manager{
		eventbus:      eventbus,
		userStates:    map[uuid.UUID]*UserState{},
		channelStates: map[uuid.UUID]*ChannelState{},
	}
	return manager
}

// GetUserState 指定したユーザーの状態を返します
func (m *Manager) GetUserState(id uuid.UUID) *UserState {
	m.statesLock.RLock()
	defer m.statesLock.RUnlock()
	s, ok := m.userStates[id]
	if !ok {
		return nil
	}
	return s.clone()
}

// GetChannelState 指定したチャンネルの状態を返します
func (m *Manager) GetChannelState(id uuid.UUID) *ChannelState {
	m.statesLock.RLock()
	defer m.statesLock.RUnlock()
	s, ok := m.channelStates[id]
	if !ok {
		return nil
	}
	return s.clone()
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
		us = &UserState{
			ConnKey: connKey,
			UserID:  user,
		}
		m.userStates[user] = us
	}

	if us.valid() && us.ChannelID != channel {
		m.channelStates[us.ChannelID].removeUser(user)
	}

	cs, ok := m.channelStates[channel]
	if !ok {
		cs = &ChannelState{
			ChannelID: channel,
			Users:     map[uuid.UUID]*UserState{},
		}
		m.channelStates[channel] = cs
	}

	us.Sessions = sessions
	us.ChannelID = channel
	cs.setUser(us)

	m.eventbus.Publish(hub.Message{
		Name: event.UserWebRTCv3StateChanged,
		Fields: hub.Fields{
			"user_id":    us.UserID,
			"channel_id": us.ChannelID,
			"sessions":   us.Sessions,
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

	if us.ConnKey != connKey {
		return ErrOccupied
	}

	delete(m.userStates, user)
	cs := m.channelStates[us.ChannelID]
	cs.removeUser(user)
	if !cs.valid() {
		delete(m.channelStates, cs.ChannelID)
	}

	m.eventbus.Publish(hub.Message{
		Name: event.UserWebRTCv3StateChanged,
		Fields: hub.Fields{
			"user_id":    us.UserID,
			"channel_id": us.ChannelID,
			"sessions":   map[string]string{},
		},
	})
	return nil
}
