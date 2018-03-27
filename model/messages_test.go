package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessage_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "messages", (&Message{}).TableName())
}

func TestMessage_Create(t *testing.T) {
	assert, _, user, channel := beforeTest(t)

	assert.Error((&Message{}).Create())
	assert.Error((&Message{UserID: user.ID}).Create())
	assert.Error((&Message{UserID: user.ID, Text: "test"}).Create())
	assert.Error((&Message{UserID: user.ID, ChannelID: channel.ID}).Create())

	m := &Message{UserID: user.ID, Text: "test", ChannelID: channel.ID}
	if assert.NoError(m.Create()) {
		assert.NotEmpty(m.ID)
		assert.NotEmpty(m.UpdaterID)
	}
}

func TestMessage_Exists(t *testing.T) {
	assert, _, user, channel := beforeTest(t)

	m := mustMakeMessage(t, user.ID, channel.ID)

	r := &Message{ID: m.ID}
	ok, err := r.Exists()

	if assert.True(ok) && assert.NoError(err) {
		assert.Equal(m.Text, r.Text)
	}

	wm := &Message{ID: CreateUUID()}
	ok, err = wm.Exists()
	assert.False(ok)
	assert.NoError(err)
}

func TestMessage_Update(t *testing.T) {
	assert, _, user, channel := beforeTest(t)

	m := mustMakeMessage(t, user.ID, channel.ID)
	m.Text = "test message"
	m.IsShared = true

	assert.NoError(m.Update())
}

func TestMessage_IsPinned(t *testing.T) {
	assert, _, user, channel := beforeTest(t)

	m := mustMakeMessage(t, user.ID, channel.ID)

	ok, err := m.IsPinned()

	if assert.NoError(err) {
		assert.False(ok)
	}

	p := &Pin{
		UserID:    user.ID,
		MessageID: m.ID,
		ChannelID: channel.ID,
	}
	require.NoError(t, p.Create())

	ok, err = m.IsPinned()

	if assert.NoError(err) {
		assert.True(ok)
	}
}

func TestGetMessagesFromChannel(t *testing.T) {
	assert, _, user, channel := beforeTest(t)

	for i := 0; i < 10; i++ {
		mustMakeMessage(t, user.ID, channel.ID)
	}

	r, err := GetMessagesByChannelID(channel.ID, 0, 0)
	if assert.NoError(err) {
		assert.Len(r, 10)
	}

	r, err = GetMessagesByChannelID(channel.ID, 3, 5)
	if assert.NoError(err) {
		assert.Len(r, 3)
	}
}

func TestGetMessage(t *testing.T) {
	assert, _, user, channel := beforeTest(t)

	m := mustMakeMessage(t, user.ID, channel.ID)

	r, err := GetMessageByID(m.ID)
	if assert.NoError(err) {
		assert.Equal(m.Text, r.Text)
	}

	_, err = GetMessageByID("wrong_id")
	assert.Error(err)
}
