package webrtc

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/utils"
)

type UserState struct {
	UserID    uuid.UUID       `json:"userId"`
	ChannelID uuid.UUID       `json:"channelId"`
	State     utils.StringSet `json:"state"`
}

func (s *UserState) Valid() bool {
	return s.UserID != uuid.Nil && s.ChannelID != uuid.Nil && len(s.State) != 0
}

func (s *UserState) Clone() *UserState {
	return &UserState{
		UserID:    s.UserID,
		ChannelID: s.ChannelID,
		State:     s.State.Clone(),
	}
}

type ChannelState struct {
	ChannelID uuid.UUID
	Users     map[uuid.UUID]*UserState
}

func (s *ChannelState) Valid() bool {
	return s.ChannelID != uuid.Nil && len(s.Users) > 0
}

func (s *ChannelState) SetUser(us *UserState) {
	s.Users[us.UserID] = us
}

func (s *ChannelState) RemoveUser(user uuid.UUID) {
	delete(s.Users, user)
}

func (s *ChannelState) Clone() *ChannelState {
	a := &ChannelState{
		ChannelID: s.ChannelID,
		Users:     map[uuid.UUID]*UserState{},
	}
	for k, v := range s.Users {
		a.Users[k] = v.Clone()
	}
	return a
}
