package ws

import (
	"testing"

	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestPingCommand(t *testing.T) {
	t.Parallel()

	session := newSession(uuid.Nil, nil, nil)
	session.commandHandler("ping")

	require.Len(t, session.send, 1)
	frame := <-session.send
	require.Equal(t, websocket.TextMessage, frame.t)
	require.JSONEq(t, `{"type":"PING","body":null}`, string(frame.data))
}

func TestWriteTypedMessageCreated(t *testing.T) {
	t.Parallel()

	streamer := NewStreamer(nil, nil, nil, zap.NewNop())
	session := newSession(uuid.Nil, nil, streamer)
	streamer.register(session)
	id := uuid.FromStringOrNil("7dd8e07f-7f5d-4331-9176-b56a4299768b")

	streamer.WriteTypedMessage(MessageCreated{ID: id}, TargetAll())
	streamer.WriteTypedMessage(MessageCreated{ID: id, IsCiting: true}, TargetAll())

	require.Len(t, session.send, 2)
	for _, want := range []string{
		`{"type":"MESSAGE_CREATED","body":{"id":"7dd8e07f-7f5d-4331-9176-b56a4299768b","is_citing":false}}`,
		`{"type":"MESSAGE_CREATED","body":{"id":"7dd8e07f-7f5d-4331-9176-b56a4299768b","is_citing":true}}`,
	} {
		frame := <-session.send
		require.Equal(t, websocket.TextMessage, frame.t)
		require.JSONEq(t, want, string(frame.data))
	}
}

func TestUnknownCommandSendsError(t *testing.T) {
	t.Parallel()

	session := newSession(uuid.Nil, nil, nil)
	session.commandHandler("unknown")

	require.Len(t, session.send, 1)
	frame := <-session.send
	require.Equal(t, websocket.TextMessage, frame.t)
	require.JSONEq(t, `{"type":"ERROR","body":"unknown command: unknown"}`, string(frame.data))
}

func TestPublicChannelCreatedOmitsDMUserID(t *testing.T) {
	t.Parallel()

	id := uuid.Must(uuid.FromString("7dd8e07f-7f5d-4331-9176-b56a4299768b"))
	msg := ChannelCreated{ID: id}.asMessage()
	require.JSONEq(t, `{"type":"CHANNEL_CREATED","body":{"id":"7dd8e07f-7f5d-4331-9176-b56a4299768b"}}`, string(msg.toJSON()))
}
