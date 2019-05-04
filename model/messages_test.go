package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMessage_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "messages", (&Message{}).TableName())
}

func TestUnread_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "unreads", (&Unread{}).TableName())
}

func TestChannelLatestMessage_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "channel_latest_messages", (&ChannelLatestMessage{}).TableName())
}

func TestArchivedMessage_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "archived_messages", (&ArchivedMessage{}).TableName())
}
