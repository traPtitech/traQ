package model

import (
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestBot_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "bots", (&Bot{}).TableName())
}

func TestBotJoinChannel_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "bot_join_channels", (&BotJoinChannel{}).TableName())
}

func TestBotEventLog_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "bot_event_logs", (&BotEventLog{}).TableName())
}

func TestBotEvent_String(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "event", BotEvent("event").String())
}

func TestBotEvents_Value(t *testing.T) {
	t.Parallel()
	es := BotEvents{"PING": true, "PONG": true}
	v, err := es.Value()
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"PING", "PONG"}, strings.Split(v.(string), " "))
}

func TestBotEvents_Scan(t *testing.T) {
	t.Parallel()

	t.Run("nil", func(t *testing.T) {
		t.Parallel()

		s := BotEvents{}
		assert.NoError(t, s.Scan(nil))
		assert.EqualValues(t, BotEvents{}, s)
	})

	t.Run("string", func(t *testing.T) {
		t.Parallel()

		s := BotEvents{}
		assert.NoError(t, s.Scan("a b c c  "))
		assert.Contains(t, s, "a")
		assert.Contains(t, s, "b")
		assert.Contains(t, s, "c")
	})

	t.Run("[]byte", func(t *testing.T) {
		t.Parallel()

		s := BotEvents{}
		assert.NoError(t, s.Scan([]byte("a b c c  ")))
		assert.Contains(t, s, "a")
		assert.Contains(t, s, "b")
		assert.Contains(t, s, "c")
	})

	t.Run("other", func(t *testing.T) {
		t.Parallel()

		s := BotEvents{}
		assert.Error(t, s.Scan(123))
	})
}

func TestBotEvents_String(t *testing.T) {
	t.Parallel()
	es := BotEvents{"PING": true, "PONG": true}
	assert.ElementsMatch(t, []string{"PING", "PONG"}, strings.Split(es.String(), " "))
}

func TestBotEvents_Contains(t *testing.T) {
	t.Parallel()
	es := BotEvents{"PING": true, "PONG": true}
	assert.True(t, es.Contains(BotEvent("PING")))
	assert.False(t, es.Contains(BotEvent("PAN")))
}

func TestBotEvents_MarshalJSON(t *testing.T) {
	t.Parallel()
	es := BotEvents{"PING": true, "PONG": true}
	b, err := es.MarshalJSON()
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{`"PING"`, `"PONG"`}, strings.Split(strings.Trim(string(b), "[]"), ","))
}
