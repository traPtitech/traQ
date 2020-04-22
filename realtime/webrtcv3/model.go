package webrtcv3

import (
	"github.com/gofrs/uuid"
)

// UserState WebRTCのユーザー状態
type UserState struct {
	ConnKey   string
	UserID    uuid.UUID
	ChannelID uuid.UUID
	Sessions  map[string]string
}

func (s *UserState) valid() bool {
	return s.ChannelID != uuid.Nil && len(s.Sessions) != 0
}

func (s *UserState) clone() *UserState {
	sessions := make(map[string]string, len(s.Sessions))
	for k, v := range s.Sessions {
		sessions[k] = v
	}
	return &UserState{
		ConnKey:   s.ConnKey,
		UserID:    s.UserID,
		ChannelID: s.ChannelID,
		Sessions:  sessions,
	}
}

// ChannelState WebRTCのチャンネル状態
type ChannelState struct {
	ChannelID uuid.UUID
	Users     map[uuid.UUID]*UserState
}

func (s *ChannelState) valid() bool {
	return s.ChannelID != uuid.Nil && len(s.Users) > 0
}

func (s *ChannelState) setUser(us *UserState) {
	s.Users[us.UserID] = us
}

func (s *ChannelState) removeUser(user uuid.UUID) {
	delete(s.Users, user)
}

func (s *ChannelState) clone() *ChannelState {
	a := &ChannelState{
		ChannelID: s.ChannelID,
		Users:     map[uuid.UUID]*UserState{},
	}
	for k, v := range s.Users {
		a.Users[k] = v.clone()
	}
	return a
}
