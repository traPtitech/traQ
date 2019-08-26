package webrtc

import (
	"errors"
	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/utils"
	"sync"
)

type Manager struct {
	eventbus      *hub.Hub
	userStates    map[uuid.UUID]*UserState
	channelStates map[uuid.UUID]*ChannelState
	statesLock    sync.RWMutex
}

func NewManager(eventbus *hub.Hub) *Manager {
	return &Manager{
		eventbus:      eventbus,
		userStates:    map[uuid.UUID]*UserState{},
		channelStates: map[uuid.UUID]*ChannelState{},
	}
}

func (m *Manager) GetUserState(id uuid.UUID) *UserState {
	m.statesLock.RLock()
	defer m.statesLock.RUnlock()
	s, ok := m.userStates[id]
	if !ok {
		return &UserState{
			UserID:    id,
			ChannelID: uuid.Nil,
			State:     utils.StringSet{},
		}
	}
	return s.Clone()
}

func (m *Manager) GetChannelState(id uuid.UUID) *ChannelState {
	m.statesLock.RLock()
	defer m.statesLock.RUnlock()
	s, ok := m.channelStates[id]
	if !ok {
		return &ChannelState{
			ChannelID: id,
			Users:     map[uuid.UUID]*UserState{},
		}
	}
	return s.Clone()
}

func (m *Manager) SetState(user, channel uuid.UUID, state utils.StringSet) error {
	if user == uuid.Nil {
		return errors.New("invalid user id")
	}
	if channel == uuid.Nil && len(state) != 0 {
		return errors.New("invalid channel id")
	}
	if channel != uuid.Nil && len(state) == 0 {
		return errors.New("invalid state")
	}

	if len(state) != 0 {
		return m.setState(user, channel, state)
	}
	return m.RemoveState(user)
}

func (m *Manager) setState(user, channel uuid.UUID, state utils.StringSet) error {
	m.statesLock.Lock()
	defer m.statesLock.Unlock()

	us, ok := m.userStates[user]
	if !ok {
		us = &UserState{
			UserID: user,
		}
		m.userStates[user] = us
	}

	if us.ChannelID != uuid.Nil && us.ChannelID != channel {
		m.channelStates[us.ChannelID].RemoveUser(user)
	}

	cs, ok := m.channelStates[channel]
	if !ok {
		cs = &ChannelState{
			ChannelID: channel,
			Users:     map[uuid.UUID]*UserState{},
		}
		m.channelStates[channel] = cs
	}

	us.State = state
	us.ChannelID = channel
	cs.SetUser(us)

	m.eventbus.Publish(hub.Message{
		Name: event.UserWebRTCStateChanged,
		Fields: hub.Fields{
			"user_id":    us.UserID,
			"channel_id": us.ChannelID,
			"state":      us.State,
		},
	})
	return nil
}

func (m *Manager) RemoveState(user uuid.UUID) error {
	m.statesLock.Lock()
	defer m.statesLock.Unlock()

	us, ok := m.userStates[user]
	if !ok {
		return nil
	}

	if us.ChannelID != uuid.Nil {
		m.channelStates[us.ChannelID].RemoveUser(user)
	}

	us.ChannelID = uuid.Nil
	us.State = utils.StringSet{}
	m.eventbus.Publish(hub.Message{
		Name: event.UserWebRTCStateChanged,
		Fields: hub.Fields{
			"user_id":    us.UserID,
			"channel_id": us.ChannelID,
			"state":      us.State,
		},
	})
	return nil
}

func (m *Manager) sweep() {
	m.statesLock.Lock()
	defer m.statesLock.Unlock()
	for k, v := range m.userStates {
		if !v.Valid() {
			delete(m.userStates, k)
		}
	}
	for k, v := range m.channelStates {
		if !v.Valid() {
			delete(m.channelStates, k)
		}
	}
}
