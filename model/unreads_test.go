package model

import (
	"github.com/satori/go.uuid"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnreadTableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "unreads", (&Unread{}).TableName())
}

func TestSetMessageUnread(t *testing.T) {
	assert, _, user, channel := beforeTest(t)
	testMessage := mustMakeMessage(t, user.ID, channel.ID)

	assert.NoError(SetMessageUnread(user.GetUID(), testMessage.GetID()))
}

func TestGetUnreadMessagesByUserID(t *testing.T) {
	assert, _, user, channel := beforeTest(t)

	for i := 0; i < 10; i++ {
		mustMakeMessageUnread(t, user.ID, mustMakeMessage(t, user.ID, channel.ID).ID)
	}

	if unreads, err := GetUnreadMessagesByUserID(user.GetUID()); assert.NoError(err) {
		assert.Len(unreads, 10)
	}
	if unreads, err := GetUnreadMessagesByUserID(uuid.Nil); assert.NoError(err) {
		assert.Len(unreads, 0)
	}
}

func TestDeleteUnreadsByMessageID(t *testing.T) {
	assert, _, user, channel := beforeTest(t)

	testMessage := mustMakeMessage(t, user.ID, channel.ID)
	testMessage2 := mustMakeMessage(t, user.ID, channel.ID)
	for i := 0; i < 10; i++ {
		mustMakeMessageUnread(t, mustMakeUser(t, "test"+strconv.Itoa(i*2)).ID, testMessage.ID)
		mustMakeMessageUnread(t, mustMakeUser(t, "test"+strconv.Itoa(i*2+1)).ID, testMessage2.ID)
	}

	if assert.NoError(DeleteUnreadsByMessageID(testMessage.GetID())) {
		count := 0
		db.Model(Unread{}).Count(&count)
		assert.Equal(10, count)
	}
	if assert.NoError(DeleteUnreadsByMessageID(testMessage2.GetID())) {
		count := 0
		db.Model(Unread{}).Count(&count)
		assert.Equal(0, count)
	}
}

func TestDeleteUnreadsByChannelID(t *testing.T) {
	assert, _, user, channel := beforeTest(t)

	creatorID := mustMakeUser(t, "creator").ID

	testMessage := mustMakeMessage(t, creatorID, channel.ID)
	mustMakeMessageUnread(t, user.ID, testMessage.ID)

	testChannel := mustMakeChannel(t, creatorID, "-unreads")
	testMessage2 := mustMakeMessage(t, creatorID, testChannel.ID)
	mustMakeMessageUnread(t, user.ID, testMessage2.ID)

	if assert.NoError(DeleteUnreadsByChannelID(channel.GetCID(), user.GetUID())) {
		count := 0
		db.Model(Unread{}).Count(&count)
		assert.Equal(1, count)
	}
}
