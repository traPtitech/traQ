package webrtc

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/utils/set"
)

// UserState WebRTCのユーザー状態
type UserState struct {
	UserID    uuid.UUID     `json:"userId"`
	ChannelID uuid.UUID     `json:"channelId"`
	State     set.StringSet `json:"state"`
}

func (s *UserState) valid() bool {
	return s.UserID != uuid.Nil && s.ChannelID != uuid.Nil && len(s.State) != 0
}

func (s *UserState) clone() *UserState {
	return &UserState{
		UserID:    s.UserID,
		ChannelID: s.ChannelID,
		State:     s.State.Clone(),
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
