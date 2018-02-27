package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

	message := &Message{UserID: user.ID, Text: "test", ChannelID: channel.ID}
	if assert.NoError(message.Create()) {
		assert.NotEmpty(message.ID)
		assert.NotEmpty(message.UpdaterID)
	}
}

func TestMessage_Update(t *testing.T) {
	assert, _, user, channel := beforeTest(t)

	message := mustMakeMessage(t, user.ID, channel.ID)
	message.Text = "nanachi"
	message.IsShared = true

	assert.NoError(message.Update())
}

func TestGetMessagesFromChannel(t *testing.T) {
	assert, _, user, channel := beforeTest(t)

	for i := 0; i < 10; i++ {
		mustMakeMessage(t, user.ID, channel.ID)
	}

	res, err := GetMessagesFromChannel(channel.ID, 0, 0)
	if assert.NoError(err) {
		assert.Len(res, 10)
	}

	res2, err := GetMessagesFromChannel(channel.ID, 3, 5)
	if assert.NoError(err) {
		assert.Len(res2, 3)
	}
}

func TestGetMessage(t *testing.T) {
	assert, _, user, channel := beforeTest(t)

	message := mustMakeMessage(t, user.ID, channel.ID)

	r, err := GetMessage(message.ID)
	if assert.NoError(err) {
		assert.Equal(message.Text, r.Text)
	}

	_, err = GetMessage("wrong_id")
	assert.Error(err)
}
