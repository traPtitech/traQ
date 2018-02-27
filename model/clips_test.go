package model

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestClip_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "clips", (&Clip{}).TableName())
}

func TestClip_Create(t *testing.T) {
	assert, _, user, channel := beforeTest(t)
	m := mustMakeMessage(t, user.ID, channel.ID)

	assert.Error((&Clip{}).Create())
	assert.Error((&Clip{UserID: user.ID}).Create())

	clip := &Clip{UserID: user.ID, MessageID: m.ID}
	assert.NoError(clip.Create())
}

func TestClip_Delete(t *testing.T) {
	assert, require, user, channel := beforeTest(t)

	message := mustMakeMessage(t, user.ID, channel.ID)

	clip := &Clip{
		UserID:    user.ID,
		MessageID: message.ID,
	}
	require.NoError(clip.Create())

	assert.Error((&Clip{}).Delete())
	assert.Error((&Clip{UserID: user.ID}).Delete())
	assert.NoError(clip.Delete())

	messageCount := 5
	for i := 0; i < messageCount; i++ {
		message := mustMakeMessage(t, user.ID, channel.ID)
		clip := &Clip{
			UserID:    user.ID,
			MessageID: message.ID,
		}
		require.NoError(clip.Create())
	}

	messages, err := GetClippedMessages(user.ID)
	assert.NoError(err)

	clip = &Clip{
		UserID:    user.ID,
		MessageID: messages[0].ID,
	}
	assert.NoError(clip.Delete())

	messages, err = GetClippedMessages(user.ID)
	if assert.NoError(err) {
		assert.Len(messages, messageCount-1)
	}
}

func TestGetClippedMessages(t *testing.T) {
	assert, require, user, channel := beforeTest(t)

	messageCount := 5
	message := mustMakeMessage(t, user.ID, channel.ID)
	clip := &Clip{
		UserID:    user.ID,
		MessageID: message.ID,
	}
	require.NoError(clip.Create())

	for i := 1; i < messageCount; i++ {
		mes := mustMakeMessage(t, user.ID, channel.ID)
		c := &Clip{
			UserID:    user.ID,
			MessageID: mes.ID,
		}
		require.NoError(c.Create())
	}

	_, err := GetClippedMessages("")
	assert.Error(err)

	messages, err := GetClippedMessages(user.ID)
	if assert.NoError(err) {
		assert.Len(messages, messageCount)
		assert.Equal(message.Text, messages[0].Text)
	}
}
