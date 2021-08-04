package model

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestBotEventType_String(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "event", BotEventType("event").String())
}

func TestBotEventTypes_Value(t *testing.T) {
	t.Parallel()
	es := BotEventTypes{"PING": struct{}{}, "PONG": struct{}{}}
	v, err := es.Value()
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"PING", "PONG"}, strings.Split(v.(string), " "))
}

func TestBotEventTypes_Scan(t *testing.T) {
	t.Parallel()

	t.Run("nil", func(t *testing.T) {
		t.Parallel()

		s := BotEventTypes{}
		assert.NoError(t, s.Scan(nil))
		assert.EqualValues(t, BotEventTypes{}, s)
	})

	t.Run("string", func(t *testing.T) {
		t.Parallel()

		s := BotEventTypes{}
		assert.NoError(t, s.Scan("a b c c  "))
		assert.Contains(t, s, BotEventType("a"))
		assert.Contains(t, s, BotEventType("b"))
		assert.Contains(t, s, BotEventType("c"))
	})

	t.Run("[]byte", func(t *testing.T) {
		t.Parallel()

		s := BotEventTypes{}
		assert.NoError(t, s.Scan([]byte("a b c c  ")))
		assert.Contains(t, s, BotEventType("a"))
		assert.Contains(t, s, BotEventType("b"))
		assert.Contains(t, s, BotEventType("c"))
	})

	t.Run("other", func(t *testing.T) {
		t.Parallel()

		s := BotEventTypes{}
		assert.Error(t, s.Scan(123))
	})
}

func TestBotEventTypes_String(t *testing.T) {
	t.Parallel()
	es := BotEventTypes{"PING": struct{}{}, "PONG": struct{}{}}
	assert.ElementsMatch(t, []string{"PING", "PONG"}, strings.Split(es.String(), " "))
}

func TestBotEventTypes_Contains(t *testing.T) {
	t.Parallel()
	es := BotEventTypes{"PING": struct{}{}, "PONG": struct{}{}}
	assert.True(t, es.Contains("PING"))
	assert.False(t, es.Contains("PAN"))
}

func TestBotEventTypes_MarshalJSON(t *testing.T) {
	t.Parallel()
	es := BotEventTypes{"PING": struct{}{}, "PONG": struct{}{}}
	b, err := es.MarshalJSON()
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{`"PING"`, `"PONG"`}, strings.Split(strings.Trim(string(b), "[]"), ","))
}
