package webrtcv3

import (
	"github.com/gofrs/uuid"
)

// UserState WebRTCのユーザー状態
type UserState interface {
	UserID() uuid.UUID
	ChannelID() uuid.UUID
	Sessions() map[string]string
}

type userState struct {
	connKey   string
	userID    uuid.UUID
	channelID uuid.UUID
	sessions  map[string]string
}

// UserID implements UserState interface.
func (s *userState) UserID() uuid.UUID {
	return s.userID
}

// ChannelID implements UserState interface.
func (s *userState) ChannelID() uuid.UUID {
	return s.channelID
}

// Sessions implements UserState interface.
func (s *userState) Sessions() map[string]string {
	return s.sessions
}

func (s *userState) valid() bool {
	return s.channelID != uuid.Nil && len(s.sessions) != 0
}

// ChannelState WebRTCのチャンネル状態
type ChannelState interface {
	ChannelID() uuid.UUID
	Users() []UserState
}

type channelState struct {
	channelID uuid.UUID
	users     map[uuid.UUID]*userState
}

// ChannelID implements ChannelState interface.
func (s *channelState) ChannelID() uuid.UUID {
	return s.channelID
}

// Users implements ChannelState interface.
func (s *channelState) Users() []UserState {
	var tmp []UserState
	for _, state := range s.users {
		tmp = append(tmp, state)
	}
	return tmp
}

func (s *channelState) valid() bool {
	return s.channelID != uuid.Nil && len(s.users) > 0
}

func (s *channelState) setUser(us *userState) {
	s.users[us.userID] = us
}

func (s *channelState) removeUser(user uuid.UUID) {
	delete(s.users, user)
}
