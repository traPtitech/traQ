package model

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestClip_TableName(t *testing.T) {
	assert.Equal(t, "clips", (&Clip{}).TableName())
}

func TestClip_Create(t *testing.T) {
	beforeTest(t)
	assert := assert.New(t)
	m := mustMakeMessage(t)

	assert.Error((&Clip{}).Create())
	assert.Error((&Clip{UserID: testUserID}).Create())

	clip := &Clip{UserID: testUserID, MessageID: m.ID}
	assert.NoError(clip.Create())
}

func TestClip_Delete(t *testing.T) {
	beforeTest(t)
	assert := assert.New(t)

	message := mustMakeMessage(t)

	clip := &Clip{
		UserID:    testUserID,
		MessageID: message.ID,
	}
	require.NoError(t, clip.Create())

	assert.Error((&Clip{}).Delete())
	assert.Error((&Clip{UserID: testUserID}).Delete())
	assert.NoError(clip.Delete())

	messageCount := 5
	for i := 0; i < messageCount; i++ {
		message := mustMakeMessage(t)
		clip := &Clip{
			UserID:    testUserID,
			MessageID: message.ID,
		}
		require.NoError(t, clip.Create())
	}

	messages, err := GetClippedMessages(testUserID)
	assert.NoError(err)

	clip = &Clip{
		UserID:    testUserID,
		MessageID: messages[0].ID,
	}
	assert.NoError(clip.Delete())

	messages, err = GetClippedMessages(testUserID)
	if assert.NoError(err) {
		assert.Len(messages, messageCount-1)
	}
}

func TestGetClippedMessages(t *testing.T) {
	beforeTest(t)
	assert := assert.New(t)

	messageCount := 5
	message := mustMakeMessage(t)
	clip := &Clip{
		UserID:    testUserID,
		MessageID: message.ID,
	}
	require.NoError(t, clip.Create())

	for i := 1; i < messageCount; i++ {
		mes := mustMakeMessage(t)
		c := &Clip{
			UserID:    testUserID,
			MessageID: mes.ID,
		}
		require.NoError(t, c.Create())
	}

	_, err := GetClippedMessages("")
	assert.Error(err)

	messages, err := GetClippedMessages(testUserID)
	if assert.NoError(err) {
		assert.Len(messages, messageCount)
		assert.Equal(message.Text, messages[0].Text)
	}
}
