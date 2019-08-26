package webrtc

import "github.com/gofrs/uuid"

type UserState struct {
	UserID    uuid.UUID
	ChannelID uuid.UUID
	State     string
}

func (s *UserState) Valid() bool {
	return s.UserID != uuid.Nil && s.ChannelID != uuid.Nil && s.State != ""
}

func (s *UserState) Clone() *UserState {
	a := *s
	return &a
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
