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

func TestMessage_Validate(t *testing.T) {
	t.Parallel()

	assert.Error(t, (&Message{Text: ""}).Validate())
	assert.NoError(t, (&Message{Text: "test"}).Validate())
}
