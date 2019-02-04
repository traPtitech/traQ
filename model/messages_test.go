package model

import (
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/utils"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMessage_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "messages", (&Message{}).TableName())
}

func TestUnread_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "unreads", (&Unread{}).TableName())
}

// TestParallelGroup7 並列テストグループ7 競合がないようなサブテストにすること
func TestParallelGroup7(t *testing.T) {
	assert, _, user, _ := beforeTest(t)

	// CreateMessage
	t.Run("TestCreateMessage", func(t *testing.T) {
		t.Parallel()

		channel := mustMakeChannelDetail(t, user.GetUID(), utils.RandAlphabetAndNumberString(20), "")

		t.Run("fail", func(t *testing.T) {
			t.Parallel()

			_, err := CreateMessage(user.GetUID(), channel.ID, "")
			assert.Error(err)
		})

		t.Run("success", func(t *testing.T) {
			t.Parallel()

			m, err := CreateMessage(user.GetUID(), channel.ID, "test")
			if assert.NoError(err) {
				assert.NotZero(m.ID)
				assert.Equal(user.ID, m.UserID.String())
				assert.Equal(channel.ID, m.ChannelID)
				assert.Equal("test", m.Text)
				assert.NotZero(m.CreatedAt)
				assert.NotZero(m.UpdatedAt)
				assert.Nil(m.DeletedAt)
			}
		})
	})

	// UpdateMessage
	t.Run("TestUpdateMessage", func(t *testing.T) {
		t.Parallel()

		channel := mustMakeChannelDetail(t, user.GetUID(), utils.RandAlphabetAndNumberString(20), "")
		m := mustMakeMessage(t, user.GetUID(), channel.ID)

		assert.Error(UpdateMessage(m.ID, ""))
		assert.NoError(UpdateMessage(m.ID, "new message"))

		m, err := GetMessageByID(m.ID)
		if assert.NoError(err) {
			assert.Equal("new message", m.Text)
		}
	})

	// DeleteMessage
	t.Run("TestDeleteMessage", func(t *testing.T) {
		t.Parallel()

		channel := mustMakeChannelDetail(t, user.GetUID(), utils.RandAlphabetAndNumberString(20), "")
		m := mustMakeMessage(t, user.GetUID(), channel.ID)

		if assert.NoError(DeleteMessage(m.ID)) {
			_, err := GetMessageByID(m.ID)
			assert.Error(err)
		}
	})

	// GetMessagesByChannelID
	t.Run("TestGetMessagesByChannelID", func(t *testing.T) {
		t.Parallel()

		channel := mustMakeChannelDetail(t, user.GetUID(), utils.RandAlphabetAndNumberString(20), "")
		for i := 0; i < 10; i++ {
			mustMakeMessage(t, user.GetUID(), channel.ID)
		}

		r, err := GetMessagesByChannelID(channel.ID, 0, 0)
		if assert.NoError(err) {
			assert.Len(r, 10)
		}

		r, err = GetMessagesByChannelID(channel.ID, 3, 5)
		if assert.NoError(err) {
			assert.Len(r, 3)
		}
	})

	//GetMessageByID
	t.Run("TestGetMessageByID", func(t *testing.T) {
		t.Parallel()

		channel := mustMakeChannelDetail(t, user.GetUID(), utils.RandAlphabetAndNumberString(20), "")
		m := mustMakeMessage(t, user.GetUID(), channel.ID)

		r, err := GetMessageByID(m.ID)
		if assert.NoError(err) {
			assert.Equal(m.Text, r.Text)
		}

		_, err = GetMessageByID(uuid.Nil)
		assert.Error(err)
	})

	// SetMessageUnread
	t.Run("TestSetMessageUnread", func(t *testing.T) {
		t.Parallel()

		user := mustMakeUser(t, utils.RandAlphabetAndNumberString(20))
		channel := mustMakeChannelDetail(t, user.GetUID(), utils.RandAlphabetAndNumberString(20), "")
		testMessage := mustMakeMessage(t, user.GetUID(), channel.ID)

		assert.NoError(SetMessageUnread(user.GetUID(), testMessage.ID))
	})

	// GetUnreadMessagesByUserID
	t.Run("TestGetUnreadMessagesByUserID", func(t *testing.T) {
		t.Parallel()

		user := mustMakeUser(t, utils.RandAlphabetAndNumberString(20))
		channel := mustMakeChannelDetail(t, user.GetUID(), utils.RandAlphabetAndNumberString(20), "")
		for i := 0; i < 10; i++ {
			mustMakeMessageUnread(t, user.GetUID(), mustMakeMessage(t, user.GetUID(), channel.ID).ID)
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
			mustMakeMessageUnread(t, mustMakeUser(t, utils.RandAlphabetAndNumberString(20)).GetUID(), testMessage.ID)
			mustMakeMessageUnread(t, mustMakeUser(t, utils.RandAlphabetAndNumberString(20)).GetUID(), testMessage2.ID)
		}

		if assert.NoError(DeleteUnreadsByMessageID(testMessage.ID)) {
			count := -1
			db.Model(Unread{}).Where(&Unread{MessageID: testMessage.ID.String()}).Count(&count)
			assert.Equal(0, count)
		}
		if assert.NoError(DeleteUnreadsByMessageID(testMessage2.ID)) {
			count := -1
			db.Model(Unread{}).Where(&Unread{MessageID: testMessage2.ID.String()}).Count(&count)
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

		mustMakeMessageUnread(t, user.GetUID(), testMessage.ID)
		mustMakeMessageUnread(t, user.GetUID(), testMessage2.ID)

		if assert.NoError(DeleteUnreadsByChannelID(channel.ID, user.GetUID())) {
			count := 0
			db.Model(Unread{}).Where(&Unread{UserID: user.ID}).Count(&count)
			assert.Equal(1, count)
		}
	})
}

func TestGetChannelLatestMessagesByUserID(t *testing.T) {
	assert, require, user, _ := beforeTest(t)

	// TODO プライベートチャンネルを考慮する
	var latests []uuid.UUID
	for j := 0; j < 10; j++ {
		ch := mustMakeChannelDetail(t, user.GetUID(), utils.RandAlphabetAndNumberString(20), "")
		if j < 5 {
			require.NoError(SubscribeChannel(user.GetUID(), ch.ID))
		}
		for i := 0; i < 10; i++ {
			mustMakeMessage(t, user.GetUID(), ch.ID)
		}
		latests = append(latests, mustMakeMessage(t, user.GetUID(), ch.ID).ID)
	}

	t.Run("SubTest1", func(t *testing.T) {
		t.Parallel()

		arr, err := GetChannelLatestMessagesByUserID(user.GetUID(), -1, false)
		derefs := make([]uuid.UUID, len(arr))
		for i := range arr {
			derefs[i] = arr[i].ID
		}
		if assert.NoError(err) {
			assert.ElementsMatch(derefs, latests)
		}
	})

	t.Run("SubTest2", func(t *testing.T) {
		t.Parallel()

		arr, err := GetChannelLatestMessagesByUserID(user.GetUID(), -1, true)
		derefs := make([]uuid.UUID, len(arr))
		for i := range arr {
			derefs[i] = arr[i].ID
		}
		if assert.NoError(err) {
			assert.ElementsMatch(derefs, latests[:5])
		}
	})

	t.Run("SubTest3", func(t *testing.T) {
		t.Parallel()

		arr, err := GetChannelLatestMessagesByUserID(user.GetUID(), 5, false)
		derefs := make([]uuid.UUID, len(arr))
		for i := range arr {
			derefs[i] = arr[i].ID
		}
		if assert.NoError(err) {
			assert.ElementsMatch(derefs, latests[5:])
		}
	})
}
