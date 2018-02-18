package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessage_TableName(t *testing.T) {
	assert.Equal(t, "messages", (&Message{}).TableName())
}

func TestMessage_Create(t *testing.T) {
	beforeTest(t)
	assert := assert.New(t)

	assert.Error((&Message{}).Create())
	assert.Error((&Message{UserID: testUserID}).Create())

	message := &Message{UserID: testUserID, Text: "test"}
	if assert.NoError(message.Create()) {
		assert.NotEmpty(message.ID)
		assert.NotEmpty(message.UpdaterID)
	}
}

func TestMessage_Update(t *testing.T) {
	beforeTest(t)
	assert := assert.New(t)

	message := mustMakeMessage(t)
	message.Text = "nanachi"
	message.IsShared = true

	assert.NoError(message.Update())
}

func TestGetMessagesFromChannel(t *testing.T) {
	beforeTest(t)
	assert := assert.New(t)

	channelID := CreateUUID()
	var messages [10]*Message

	for i := 0; i < 10; i++ {
		messages[i] = &Message{
			UserID:    testUserID,
			ChannelID: channelID,
			Text:      "popopo",
		}
		require.NoError(t, messages[i].Create())
		time.Sleep(1500 * time.Millisecond)
	}

	res, err := GetMessagesFromChannel(channelID, 0, 0)
	if assert.NoError(err) {
		assert.Len(res, 10)
	}

	for i := 0; i < 10; i++ {
		assert.Equal(res[i].ID, messages[9-i].ID, "message is not ordered by createdAt")
	}

	res2, err := GetMessagesFromChannel(channelID, 3, 5)
	if assert.NoError(err) {
		assert.Len(res2, 3)
		assert.Equal(messages[4].ID, res2[0].ID)
	}
}

func TestGetMessage(t *testing.T) {
	beforeTest(t)
	assert := assert.New(t)

	message := mustMakeMessage(t)

	var r *Message
	r, err := GetMessage(message.ID)
	if assert.NoError(err) {
		assert.Equal(message.Text, r.Text)
	}

	_, err = GetMessage("wrong_id")
	assert.Error(err)
}
