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
	testMessage := mustMakeMessage(t, user.GetUID(), channel.GetCID())

	assert.NoError(SetMessageUnread(user.GetUID(), testMessage.GetID()))
}

func TestGetUnreadMessagesByUserID(t *testing.T) {
	assert, _, user, channel := beforeTest(t)

	for i := 0; i < 10; i++ {
		mustMakeMessageUnread(t, user.GetUID(), mustMakeMessage(t, user.GetUID(), channel.GetCID()).GetID())
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

	testMessage := mustMakeMessage(t, user.GetUID(), channel.GetCID())
	testMessage2 := mustMakeMessage(t, user.GetUID(), channel.GetCID())
	for i := 0; i < 10; i++ {
		mustMakeMessageUnread(t, mustMakeUser(t, "test"+strconv.Itoa(i*2)).GetUID(), testMessage.GetID())
		mustMakeMessageUnread(t, mustMakeUser(t, "test"+strconv.Itoa(i*2+1)).GetUID(), testMessage2.GetID())
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

	creator := mustMakeUser(t, "creator")

	testMessage := mustMakeMessage(t, creator.GetUID(), channel.GetCID())
	mustMakeMessageUnread(t, user.GetUID(), testMessage.GetID())

	testChannel := mustMakeChannel(t, creator.GetUID(), "-unreads")
	testMessage2 := mustMakeMessage(t, creator.GetUID(), testChannel.GetCID())
	mustMakeMessageUnread(t, user.GetUID(), testMessage2.GetID())

	if assert.NoError(DeleteUnreadsByChannelID(channel.GetCID(), user.GetUID())) {
		count := 0
		db.Model(Unread{}).Count(&count)
		assert.Equal(1, count)
	}
}
