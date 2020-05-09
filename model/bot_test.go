package model

import (
	"github.com/stretchr/testify/assert"
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
