package model

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnreadTableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "unreads", (&Unread{}).TableName())
}

func beforeUnreadsTest(t *testing.T, userID, channelID string) (*Unread, *Unread, *Unread) {
	testMessage := mustMakeMessage(t, userID, channelID)

	testUnread := &Unread{
		UserID:    userID,
		MessageID: testMessage.ID,
	}
	emptyUserIDUnread := &Unread{
		MessageID: testMessage.ID,
	}
	emptyMessageIDUnread := &Unread{
		UserID: userID,
	}
	return testUnread, emptyUserIDUnread, emptyMessageIDUnread
}

func TestUnreadCreate(t *testing.T) {
	assert, require, user, channel := beforeTest(t)
	testUnread, emptyUserIDUnread, emptyMessageIDUnread := beforeUnreadsTest(t, user.ID, channel.ID)

	// 正常系
	assert.NoError(testUnread.Create())
	unreads, err := GetUnreadsByUserID(testUnread.UserID)
	require.NoError(err)
	assert.Len(unreads, 1)
	assert.Equal(*unreads[0], *testUnread)

	// 異常系
	assert.Error(emptyUserIDUnread.Create())
	assert.Error(emptyMessageIDUnread.Create())
}

func TestUnreadDelete(t *testing.T) {
	assert, require, user, channel := beforeTest(t)
	testUnread, emptyUserIDUnread, emptyMessageIDUnread := beforeUnreadsTest(t, user.ID, channel.ID)

	// 正常系
	require.NoError(testUnread.Create())
	assert.NoError(testUnread.Delete())
	unreads, err := GetUnreadsByUserID(testUnread.UserID)
	require.NoError(err)
	assert.Len(unreads, 0)

	// 異常系
	assert.Error(emptyUserIDUnread.Delete())
	assert.Error(emptyMessageIDUnread.Delete())
}

func TestGetUnreadsByUserID(t *testing.T) {
	assert, require, user, channel := beforeTest(t)
	testUnread, _, _ := beforeUnreadsTest(t, user.ID, channel.ID)

	// 正常系
	require.NoError(testUnread.Create())
	unreads, err := GetUnreadsByUserID(testUnread.UserID)
	assert.NoError(err)
	assert.Len(unreads, 1)
	assert.Equal(*unreads[0], *testUnread)

	// 異常系
	_, emptyErr := GetUnreadsByUserID("")
	assert.Error(emptyErr)
	nobodyUnreads, nobodyErr := GetUnreadsByUserID(nobodyID)
	assert.NoError(nobodyErr)
	assert.Len(nobodyUnreads, 0)
}

func TestDeleteUnreadsByMessageID(t *testing.T) {
	assert, _, user, channel := beforeTest(t)

	testMessage := mustMakeMessage(t, user.ID, channel.ID)
	testMessage2 := mustMakeMessage(t, user.ID, channel.ID)

	for i := 0; i < 10; i++ {
		mustMakeMessageUnread(t, mustMakeUser(t, "test"+strconv.Itoa(i*2)).ID, testMessage.ID)
		mustMakeMessageUnread(t, mustMakeUser(t, "test"+strconv.Itoa(i*2+1)).ID, testMessage2.ID)
	}

	// 正常系
	if assert.NoError(DeleteUnreadsByMessageID(testMessage.ID)) {
		if n, err := db.Count(&Unread{}); assert.NoError(err) {
			assert.EqualValues(10, n)
		}
	}
	if assert.NoError(DeleteUnreadsByMessageID(testMessage2.ID)) {
		if n, err := db.Count(&Unread{}); assert.NoError(err) {
			assert.EqualValues(0, n)
		}
	}

	// 異常系
	assert.Error(DeleteUnreadsByMessageID(""))
}

func TestDeleteUnreadsByChannelID(t *testing.T) {
	assert, _, user, channel := beforeTest(t)

	creatorID := mustMakeUser(t, "creator").ID

	testMessage := mustMakeMessage(t, creatorID, channel.ID)
	mustMakeMessageUnread(t, user.ID, testMessage.ID)

	testChannel := mustMakeChannel(t, creatorID, "-unreads")
	testMessage2 := mustMakeMessage(t, creatorID, testChannel.ID)
	mustMakeMessageUnread(t, user.ID, testMessage2.ID)

	if assert.NoError(DeleteUnreadsByChannelID(channel.ID, user.ID)) {
		if n, err := db.Count(&Unread{}); assert.NoError(err) {
			assert.EqualValues(1, n)
		}
	}
}
