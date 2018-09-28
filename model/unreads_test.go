package model

import (
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/utils"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnreadTableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "unreads", (&Unread{}).TableName())
}

// TestParallelGroup9 並列テストグループ9 競合がないようなサブテストにすること
func TestParallelGroup9(t *testing.T) {
	assert, _, _, _ := beforeTest(t)

	// SetMessageUnread
	t.Run("TestSetMessageUnread", func(t *testing.T) {
		t.Parallel()

		user := mustMakeUser(t, utils.RandAlphabetAndNumberString(20))
		channel := mustMakeChannelDetail(t, user.GetUID(), utils.RandAlphabetAndNumberString(20), "")
		testMessage := mustMakeMessage(t, user.GetUID(), channel.ID)

		assert.NoError(SetMessageUnread(user.GetUID(), testMessage.GetID()))
	})

	// GetUnreadMessagesByUserID
	t.Run("TestGetUnreadMessagesByUserID", func(t *testing.T) {
		t.Parallel()

		user := mustMakeUser(t, utils.RandAlphabetAndNumberString(20))
		channel := mustMakeChannelDetail(t, user.GetUID(), utils.RandAlphabetAndNumberString(20), "")
		for i := 0; i < 10; i++ {
			mustMakeMessageUnread(t, user.GetUID(), mustMakeMessage(t, user.GetUID(), channel.ID).GetID())
		}

		if unreads, err := GetUnreadMessagesByUserID(user.GetUID()); assert.NoError(err) {
			assert.Len(unreads, 10)
		}
		if unreads, err := GetUnreadMessagesByUserID(uuid.Nil); assert.NoError(err) {
			assert.Len(unreads, 0)
		}
	})

	// DeleteUnreadsByMessageID
	t.Run("TestDeleteUnreadsByMessageID", func(t *testing.T) {
		t.Parallel()

		user := mustMakeUser(t, utils.RandAlphabetAndNumberString(20))
		channel := mustMakeChannelDetail(t, user.GetUID(), utils.RandAlphabetAndNumberString(20), "")
		testMessage := mustMakeMessage(t, user.GetUID(), channel.ID)
		testMessage2 := mustMakeMessage(t, user.GetUID(), channel.ID)
		for i := 0; i < 10; i++ {
			mustMakeMessageUnread(t, mustMakeUser(t, utils.RandAlphabetAndNumberString(20)).GetUID(), testMessage.GetID())
			mustMakeMessageUnread(t, mustMakeUser(t, utils.RandAlphabetAndNumberString(20)).GetUID(), testMessage2.GetID())
		}

		if assert.NoError(DeleteUnreadsByMessageID(testMessage.GetID())) {
			count := -1
			db.Model(Unread{}).Where(&Unread{MessageID: testMessage.ID}).Count(&count)
			assert.Equal(0, count)
		}
		if assert.NoError(DeleteUnreadsByMessageID(testMessage2.GetID())) {
			count := -1
			db.Model(Unread{}).Where(&Unread{MessageID: testMessage2.ID}).Count(&count)
			assert.Equal(0, count)
		}
	})

	// DeleteUnreadsByChannelID
	t.Run("TestDeleteUnreadsByChannelID", func(t *testing.T) {
		t.Parallel()

		creator := mustMakeUser(t, utils.RandAlphabetAndNumberString(20))
		user := mustMakeUser(t, utils.RandAlphabetAndNumberString(20))
		channel := mustMakeChannelDetail(t, creator.GetUID(), utils.RandAlphabetAndNumberString(20), "")
		channel2 := mustMakeChannelDetail(t, creator.GetUID(), utils.RandAlphabetAndNumberString(20), "")
		testMessage := mustMakeMessage(t, creator.GetUID(), channel.ID)
		testMessage2 := mustMakeMessage(t, creator.GetUID(), channel2.ID)

		mustMakeMessageUnread(t, user.GetUID(), testMessage.GetID())
		mustMakeMessageUnread(t, user.GetUID(), testMessage2.GetID())

		if assert.NoError(DeleteUnreadsByChannelID(channel.ID, user.GetUID())) {
			count := 0
			db.Model(Unread{}).Where(&Unread{UserID: user.ID}).Count(&count)
			assert.Equal(1, count)
		}
	})
}
