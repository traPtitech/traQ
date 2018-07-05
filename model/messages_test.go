package model

import (
	"github.com/satori/go.uuid"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMessage_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "messages", (&Message{}).TableName())
}

func TestCreateMessage(t *testing.T) {
	assert, _, user, channel := beforeTest(t)

	_, err := CreateMessage(user.GetUID(), channel.GetCID(), "")
	assert.Error(err)

	m, err := CreateMessage(user.GetUID(), channel.GetCID(), "test")
	if assert.NoError(err) {
		assert.NotEmpty(m.ID)
		assert.Equal(user.ID, m.UserID)
		assert.Equal(channel.ID, m.ChannelID)
		assert.Equal("test", m.Text)
		assert.NotZero(m.CreatedAt)
		assert.NotZero(m.UpdatedAt)
		assert.Nil(m.DeletedAt)
	}
}

func TestUpdateMessage(t *testing.T) {
	assert, _, user, channel := beforeTest(t)

	m := mustMakeMessage(t, user.GetUID(), channel.GetCID())

	assert.Error(UpdateMessage(m.GetID(), ""))
	assert.NoError(UpdateMessage(m.GetID(), "new message"))

	m, err := GetMessageByID(m.GetID())
	if assert.NoError(err) {
		assert.Equal("new message", m.Text)
	}
}

func TestDeleteMessage(t *testing.T) {
	assert, _, user, channel := beforeTest(t)

	m := mustMakeMessage(t, user.GetUID(), channel.GetCID())

	if assert.NoError(DeleteMessage(m.GetID())) {
		_, err := GetMessageByID(m.GetID())
		assert.Error(err)
	}
}

func TestGetMessagesByChannelID(t *testing.T) {
	assert, _, user, channel := beforeTest(t)

	for i := 0; i < 10; i++ {
		mustMakeMessage(t, user.GetUID(), channel.GetCID())
	}

	r, err := GetMessagesByChannelID(channel.GetCID(), 0, 0)
	if assert.NoError(err) {
		assert.Len(r, 10)
	}

	r, err = GetMessagesByChannelID(channel.GetCID(), 3, 5)
	if assert.NoError(err) {
		assert.Len(r, 3)
	}
}

func TestGetMessageByID(t *testing.T) {
	assert, _, user, channel := beforeTest(t)

	m := mustMakeMessage(t, user.GetUID(), channel.GetCID())

	r, err := GetMessageByID(m.GetID())
	if assert.NoError(err) {
		assert.Equal(m.Text, r.Text)
	}

	_, err = GetMessageByID(uuid.Nil)
	assert.Error(err)
}
