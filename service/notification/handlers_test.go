package notification

import (
	"encoding/json"
	"testing"
	"testing/synctest"
	"time"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"github.com/leandro-lugaresi/hub"
	"github.com/stretchr/testify/require"

	"github.com/traPtitech/traQ/service/channel/mock_channel"
	"github.com/traPtitech/traQ/service/qall"
	"github.com/traPtitech/traQ/service/ws"
)

type messageWriterFunc func(ws.TypedMessage, ws.TargetFunc)

func (f messageWriterFunc) WriteTypedMessage(msg ws.TypedMessage, target ws.TargetFunc) {
	f(msg, target)
}

type recipientSession struct {
	ws.Session
	id uuid.UUID
}

func (s recipientSession) UserID() uuid.UUID {
	return s.id
}

func TestChannelCreatedDeliversOtherDMMember(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		channelID := uuid.Must(uuid.NewV4())
		first := uuid.Must(uuid.NewV4())
		second := uuid.Must(uuid.NewV4())
		manager := mock_channel.NewMockManager(gomock.NewController(t))
		manager.EXPECT().GetDMChannelMembers(gomock.Any(), channelID).Return([]uuid.UUID{first, second}, nil)

		type delivery struct {
			msg    ws.TypedMessage
			target ws.TargetFunc
		}
		deliveries := make(chan delivery, 2)
		ns := &Service{
			cm: manager,
			messageWriter: messageWriterFunc(func(msg ws.TypedMessage, target ws.TargetFunc) {
				deliveries <- delivery{msg, target}
			}),
		}
		channelCreatedHandler(ns, hub.Message{Fields: hub.Fields{"channel_id": channelID, "private": true}})
		synctest.Wait()

		require.Len(t, deliveries, 2)
		expected := map[uuid.UUID]uuid.UUID{first: second, second: first}
		for range 2 {
			sent := <-deliveries
			body, ok := sent.msg.(ws.ChannelCreated)
			require.True(t, ok)
			require.Equal(t, channelID, body.ID)
			require.NotNil(t, body.DMUserID)
			recipient, ok := expected[*body.DMUserID]
			require.True(t, ok)
			require.True(t, sent.target(recipientSession{id: recipient}))
			require.False(t, sent.target(recipientSession{id: *body.DMUserID}))
			delete(expected, *body.DMUserID)
		}
	})
}

func TestQallRoomStateChangedKeepsPayloadShape(t *testing.T) {
	for _, tc := range []struct {
		name  string
		rooms []qall.RoomWithParticipants
	}{
		{name: "nil rooms"},
		{name: "empty rooms", rooms: []qall.RoomWithParticipants{}},
		{name: "room contents", rooms: []qall.RoomWithParticipants{
			{Participants: []qall.Participant{}},
			{Metadata: new(""), Participants: nil},
			{IsWebinar: true, Participants: []qall.Participant{
				{Identity: "user", JoinedAt: time.Date(2026, 11, 4, 5, 0, 0, 0, time.UTC), CanPublish: true},
				{Attributes: new(map[string]string{})},
				{Attributes: new(map[string]string(nil))},
			}},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				deliveries := make(chan ws.TypedMessage, 1)
				ns := &Service{messageWriter: messageWriterFunc(func(msg ws.TypedMessage, _ ws.TargetFunc) {
					deliveries <- msg
				})}
				qallRoomStateChangedHandler(ns, hub.Message{Fields: hub.Fields{"roomStates": tc.rooms}})
				synctest.Wait()

				require.Len(t, deliveries, 1)
				payload := <-deliveries
				require.IsType(t, ws.QallRoomStateChanged{}, payload)
				got, err := json.Marshal(payload)
				require.NoError(t, err)
				want, err := json.Marshal(map[string]any{"roomStates": tc.rooms})
				require.NoError(t, err)
				require.JSONEq(t, string(want), string(got))
			})
		})
	}
}
