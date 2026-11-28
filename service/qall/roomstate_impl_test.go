package qall

import (
	"sync"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/livekit/protocol/livekit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestRepository() *Repository {
	return NewRoomStateManager("", "", "", nil).(*Repository)
}

func TestParseMetadata(t *testing.T) {
	t.Parallel()

	assert.Equal(t, Metadata{}, ParseMetadata(""))
	assert.Equal(t, Metadata{}, ParseMetadata("not json"))
	assert.Equal(t, Metadata{Status: "hello", IsWebinar: true}, ParseMetadata(`{"status":"hello","isWebinar":true}`))
}

func TestParticipantFromLiveKit(t *testing.T) {
	t.Parallel()

	p := participantFromLiveKit(&livekit.ParticipantInfo{
		Identity:   "user_1",
		Name:       "user",
		JoinedAt:   100,
		Permission: &livekit.ParticipantPermission{CanPublish: true},
	})
	assert.True(t, p.CanPublish)
	require.NotNil(t, p.Attributes)
	assert.Empty(t, *p.Attributes)

	p = participantFromLiveKit(&livekit.ParticipantInfo{Identity: "user_2"})
	assert.False(t, p.CanPublish)
}

func TestRepository_AddParticipantToRoomState_Merge(t *testing.T) {
	t.Parallel()

	roomID := uuid.Must(uuid.NewV4())
	room := &livekit.Room{Name: roomID.String(), Metadata: `{"status":"","isWebinar":true}`}

	t.Run("adds the room when it is unknown", func(t *testing.T) {
		t.Parallel()
		r := newTestRepository()
		r.AddParticipantToRoomState(room, &livekit.ParticipantInfo{Identity: "a", Name: "a"})

		state := r.GetRoomState(roomID.String())
		require.NotNil(t, state)
		assert.True(t, state.IsWebinar)
		assert.Len(t, state.Participants, 1)
	})

	t.Run("does not duplicate the same identity", func(t *testing.T) {
		t.Parallel()
		r := newTestRepository()
		r.AddParticipantToRoomState(room, &livekit.ParticipantInfo{Identity: "a", Name: "a"})
		r.AddParticipantToRoomState(room, &livekit.ParticipantInfo{
			Identity:   "a",
			Name:       "a",
			Permission: &livekit.ParticipantPermission{CanPublish: true},
		})

		state := r.GetRoomState(roomID.String())
		require.NotNil(t, state)
		require.Len(t, state.Participants, 1)
		assert.True(t, state.Participants[0].CanPublish)
	})

	t.Run("ignores rooms that are not channel ids", func(t *testing.T) {
		t.Parallel()
		r := newTestRepository()
		r.AddParticipantToRoomState(&livekit.Room{Name: "not-a-uuid"}, &livekit.ParticipantInfo{Identity: "a"})
		assert.Empty(t, r.GetState())
	})
}

func TestRepository_AddRoomState_KeepsParticipants(t *testing.T) {
	t.Parallel()

	r := newTestRepository()
	roomID := uuid.Must(uuid.NewV4())
	r.AddParticipantToRoomState(&livekit.Room{Name: roomID.String()}, &livekit.ParticipantInfo{Identity: "a"})

	r.AddRoomState(RoomWithParticipants{RoomID: roomID})
	state := r.GetState()
	require.Len(t, state, 1)
	assert.Len(t, state[0].Participants, 1)
}

func TestRepository_RemoveParticipantAndRoom(t *testing.T) {
	t.Parallel()

	r := newTestRepository()
	roomID := uuid.Must(uuid.NewV4())
	room := &livekit.Room{Name: roomID.String()}
	r.AddParticipantToRoomState(room, &livekit.ParticipantInfo{Identity: "a"})
	r.AddParticipantToRoomState(room, &livekit.ParticipantInfo{Identity: "b"})

	r.RemoveParticipant(roomID.String(), "a")
	state := r.GetRoomState(roomID.String())
	require.NotNil(t, state)
	require.Len(t, state.Participants, 1)
	assert.Equal(t, "b", state.Participants[0].Identity)

	r.RemoveRoomState(roomID.String())
	assert.Nil(t, r.GetRoomState(roomID.String()))
}

func TestRepository_GetStateReturnsCopy(t *testing.T) {
	t.Parallel()

	r := newTestRepository()
	roomID := uuid.Must(uuid.NewV4())
	r.AddParticipantToRoomState(&livekit.Room{Name: roomID.String()}, &livekit.ParticipantInfo{
		Identity:   "a",
		Attributes: map[string]string{"k": "v"},
	})

	state := r.GetState()
	state[0].Participants[0].Identity = "changed"
	(*state[0].Participants[0].Attributes)["k"] = "changed"

	room := r.GetRoomState(roomID.String())
	require.NotNil(t, room)
	assert.Equal(t, "a", room.Participants[0].Identity)
	assert.Equal(t, "v", (*room.Participants[0].Attributes)["k"])
}

func TestRepository_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	r := newTestRepository()
	roomID := uuid.Must(uuid.NewV4())
	room := &livekit.Room{Name: roomID.String()}

	var wg sync.WaitGroup
	for i := range 50 {
		identity := uuid.Must(uuid.NewV4()).String()
		wg.Go(func() {
			r.AddParticipantToRoomState(room, &livekit.ParticipantInfo{Identity: identity})
			_ = r.GetState()
			r.UpdateParticipantCanPublish(roomID.String(), identity, i%2 == 0)
			_ = r.GetRoomState(roomID.String())
		})
	}
	wg.Wait()

	state := r.GetRoomState(roomID.String())
	require.NotNil(t, state)
	assert.Len(t, state.Participants, 50)
}

func TestRoomWithParticipants_CanPublish(t *testing.T) {
	t.Parallel()

	owner := uuid.Must(uuid.NewV4())
	listener := uuid.Must(uuid.NewV4())

	t.Run("everyone can publish outside webinars", func(t *testing.T) {
		t.Parallel()
		room := RoomWithParticipants{IsWebinar: false}
		assert.True(t, room.CanPublish(listener))
	})

	t.Run("only publishers can publish in webinars", func(t *testing.T) {
		t.Parallel()
		room := RoomWithParticipants{IsWebinar: true, Publishers: []uuid.UUID{owner}}
		assert.True(t, room.CanPublish(owner))
		assert.False(t, room.CanPublish(listener))
	})

	t.Run("publishers keep the permission after leaving", func(t *testing.T) {
		t.Parallel()
		room := RoomWithParticipants{IsWebinar: true, Publishers: []uuid.UUID{owner}, Participants: []Participant{}}
		assert.True(t, room.CanPublish(owner))
	})

	t.Run("older webinars fall back to the participants", func(t *testing.T) {
		t.Parallel()
		room := RoomWithParticipants{
			IsWebinar: true,
			Participants: []Participant{
				{Name: owner.String(), CanPublish: true},
				{Name: listener.String(), CanPublish: false},
			},
		}
		assert.True(t, room.CanPublish(owner))
		assert.False(t, room.CanPublish(listener))
	})
}

func TestRoomStateFromLiveKit(t *testing.T) {
	t.Parallel()

	roomID := uuid.Must(uuid.NewV4())
	owner := uuid.Must(uuid.NewV4())
	room := RoomStateFromLiveKit(roomID, `{"status":"hi","isWebinar":true,"publishers":["`+owner.String()+`"]}`)

	assert.Equal(t, roomID, room.RoomID)
	assert.True(t, room.IsWebinar)
	require.NotNil(t, room.Metadata)
	assert.Equal(t, "hi", *room.Metadata)
	assert.Equal(t, []uuid.UUID{owner}, room.Publishers)
	assert.Equal(t, Metadata{Status: "hi", IsWebinar: true, Publishers: []uuid.UUID{owner}}, room.RoomMetadata())
}

func TestRepository_UpdateRoomMetadataKeepsPublishers(t *testing.T) {
	t.Parallel()

	r := newTestRepository()
	roomID := uuid.Must(uuid.NewV4())
	owner := uuid.Must(uuid.NewV4())
	r.AddRoomState(RoomStateFromLiveKit(roomID, `{"isWebinar":true,"publishers":["`+owner.String()+`"]}`))

	other := uuid.Must(uuid.NewV4())
	r.UpdateRoomMetadata(roomID.String(), Metadata{Status: "s", IsWebinar: true, Publishers: []uuid.UUID{owner, other}})

	room := r.GetRoomState(roomID.String())
	require.NotNil(t, room)
	assert.Equal(t, []uuid.UUID{owner, other}, room.Publishers)
	assert.Equal(t, "s", *room.Metadata)
}
