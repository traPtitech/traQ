package repository

import (
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/traPtitech/traQ/model"
	"testing"
)

func TestRepositoryImpl_CreateMessage(t *testing.T) {
	t.Parallel()
	repo, _, _, user, channel := setupWithUserAndChannel(t, common)

	t.Run("failures", func(t *testing.T) {
		t.Parallel()

		_, err := repo.CreateMessage(user.ID, channel.ID, "")
		assert.Error(t, err)

		_, err = repo.CreateMessage(user.ID, uuid.Nil, "a")
		assert.Error(t, err)

		_, err = repo.CreateMessage(uuid.Nil, channel.ID, "a")
		assert.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		m, err := repo.CreateMessage(user.ID, channel.ID, "test")
		if assert.NoError(err) {
			assert.NotZero(m.ID)
			assert.Equal(user.ID, m.UserID)
			assert.Equal(channel.ID, m.ChannelID)
			assert.Equal("test", m.Text)
			assert.NotZero(m.CreatedAt)
			assert.NotZero(m.UpdatedAt)
			assert.Nil(m.DeletedAt)
		}
	})
}

func TestRepositoryImpl_UpdateMessage(t *testing.T) {
	t.Parallel()
	repo, assert, _, user, channel := setupWithUserAndChannel(t, common)

	m := mustMakeMessage(t, repo, user.ID, channel.ID)
	originalText := m.Text

	assert.Error(repo.UpdateMessage(m.ID, ""))
	assert.EqualError(repo.UpdateMessage(uuid.Must(uuid.NewV4()), "new message"), ErrNotFound.Error())
	assert.EqualError(repo.UpdateMessage(uuid.Nil, "new message"), ErrNilID.Error())
	assert.NoError(repo.UpdateMessage(m.ID, "new message"))

	m, err := repo.GetMessageByID(m.ID)
	if assert.NoError(err) {
		assert.Equal("new message", m.Text)
		assert.Equal(1, count(t, getDB(repo).Model(&model.ArchivedMessage{}).Where(&model.ArchivedMessage{MessageID: m.ID, Text: originalText})))
	}
}

func TestRepositoryImpl_DeleteMessage(t *testing.T) {
	t.Parallel()
	repo, assert, _, user, channel := setupWithUserAndChannel(t, common)

	m := mustMakeMessage(t, repo, user.ID, channel.ID)

	assert.EqualError(repo.DeleteMessage(uuid.Nil), ErrNilID.Error())

	if assert.NoError(repo.DeleteMessage(m.ID)) {
		_, err := repo.GetMessageByID(m.ID)
		assert.EqualError(err, ErrNotFound.Error())
	}
	assert.EqualError(repo.DeleteMessage(m.ID), ErrNotFound.Error())
}

func TestRepositoryImpl_GetMessagesByChannelID(t *testing.T) {
	t.Parallel()
	repo, assert, _, user, channel := setupWithUserAndChannel(t, common)

	for i := 0; i < 10; i++ {
		mustMakeMessage(t, repo, user.ID, channel.ID)
	}

	r, err := repo.GetMessagesByChannelID(channel.ID, 0, 0)
	if assert.NoError(err) {
		assert.Len(r, 10)
	}

	r, err = repo.GetMessagesByChannelID(channel.ID, 3, 5)
	if assert.NoError(err) {
		assert.Len(r, 3)
	}

	r, err = repo.GetMessagesByChannelID(uuid.Nil, 0, 0)
	if assert.NoError(err) {
		assert.Len(r, 0)
	}
}

func TestRepositoryImpl_GetMessagesByUserID(t *testing.T) {
	t.Parallel()
	repo, assert, _, user, channel := setupWithUserAndChannel(t, common)

	for i := 0; i < 10; i++ {
		mustMakeMessage(t, repo, user.ID, channel.ID)
	}

	r, err := repo.GetMessagesByUserID(user.ID, 0, 0)
	if assert.NoError(err) {
		assert.Len(r, 10)
	}

	r, err = repo.GetMessagesByUserID(user.ID, 3, 5)
	if assert.NoError(err) {
		assert.Len(r, 3)
	}

	r, err = repo.GetMessagesByUserID(uuid.Nil, 0, 0)
	if assert.NoError(err) {
		assert.Len(r, 0)
	}
}

func TestRepositoryImpl_GetMessageByID(t *testing.T) {
	t.Parallel()
	repo, assert, _, user, channel := setupWithUserAndChannel(t, common)

	m := mustMakeMessage(t, repo, user.ID, channel.ID)

	r, err := repo.GetMessageByID(m.ID)
	if assert.NoError(err) {
		assert.Equal(m.Text, r.Text)
	}

	_, err = repo.GetMessageByID(uuid.Nil)
	assert.Error(err)

	_, err = repo.GetMessageByID(uuid.Must(uuid.NewV4()))
	assert.Error(err)
}

func TestRepositoryImpl_SetMessageUnread(t *testing.T) {
	t.Parallel()
	repo, assert, _, user, channel := setupWithUserAndChannel(t, common)

	m := mustMakeMessage(t, repo, user.ID, channel.ID)

	assert.Error(repo.SetMessageUnread(uuid.Nil, m.ID))
	assert.Error(repo.SetMessageUnread(user.ID, uuid.Nil))
	if assert.NoError(repo.SetMessageUnread(user.ID, m.ID)) {
		assert.Equal(1, count(t, getDB(repo).Model(model.Unread{}).Where(model.Unread{UserID: user.ID})))
	}
	assert.NoError(repo.SetMessageUnread(user.ID, m.ID))
}

func TestRepositoryImpl_GetUnreadMessagesByUserID(t *testing.T) {
	t.Parallel()
	repo, assert, _, user, channel := setupWithUserAndChannel(t, common)

	for i := 0; i < 10; i++ {
		mustMakeMessageUnread(t, repo, user.ID, mustMakeMessage(t, repo, user.ID, channel.ID).ID)
	}

	if unreads, err := repo.GetUnreadMessagesByUserID(user.ID); assert.NoError(err) {
		assert.Len(unreads, 10)
	}
	if unreads, err := repo.GetUnreadMessagesByUserID(uuid.Nil); assert.NoError(err) {
		assert.Len(unreads, 0)
	}
}

func TestRepositoryImpl_DeleteUnreadsByMessageID(t *testing.T) {
	t.Parallel()
	repo, _, _, user, channel := setupWithUserAndChannel(t, common)

	testMessage := mustMakeMessage(t, repo, user.ID, channel.ID)
	testMessage2 := mustMakeMessage(t, repo, user.ID, channel.ID)
	for i := 0; i < 10; i++ {
		mustMakeMessageUnread(t, repo, mustMakeUser(t, repo, random).ID, testMessage.ID)
		mustMakeMessageUnread(t, repo, mustMakeUser(t, repo, random).ID, testMessage2.ID)
	}

	t.Run("nil id", func(t *testing.T) {
		t.Parallel()

		assert.EqualError(t, repo.DeleteUnreadsByMessageID(uuid.Nil), ErrNilID.Error())
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		if assert.NoError(repo.DeleteUnreadsByMessageID(testMessage.ID)) {
			assert.Equal(0, count(t, getDB(repo).Model(model.Unread{}).Where(&model.Unread{MessageID: testMessage.ID})))
		}
		if assert.NoError(repo.DeleteUnreadsByMessageID(testMessage2.ID)) {
			assert.Equal(0, count(t, getDB(repo).Model(model.Unread{}).Where(&model.Unread{MessageID: testMessage2.ID})))
		}
	})
}

func TestRepositoryImpl_DeleteUnreadsByChannelID(t *testing.T) {
	t.Parallel()
	repo, _, _, user, channel := setupWithUserAndChannel(t, common)

	creator := mustMakeUser(t, repo, random)
	channel2 := mustMakeChannel(t, repo, random)
	testMessage := mustMakeMessage(t, repo, creator.ID, channel.ID)
	testMessage2 := mustMakeMessage(t, repo, creator.ID, channel2.ID)
	mustMakeMessageUnread(t, repo, user.ID, testMessage.ID)
	mustMakeMessageUnread(t, repo, user.ID, testMessage2.ID)

	t.Run("nil id", func(t *testing.T) {
		t.Parallel()

		assert.EqualError(t, repo.DeleteUnreadsByChannelID(channel.ID, uuid.Nil), ErrNilID.Error())
		assert.EqualError(t, repo.DeleteUnreadsByChannelID(uuid.Nil, user.ID), ErrNilID.Error())
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		if assert.NoError(repo.DeleteUnreadsByChannelID(channel.ID, user.ID)) {
			assert.Equal(1, count(t, getDB(repo).Model(model.Unread{}).Where(&model.Unread{UserID: user.ID})))
		}
	})
}

func TestRepositoryImpl_GetChannelLatestMessagesByUserID(t *testing.T) {
	t.Parallel()
	repo, _, require, user := setupWithUser(t, ex1)

	var latests []uuid.UUID
	for j := 0; j < 10; j++ {
		ch := mustMakeChannel(t, repo, random)
		if j < 5 {
			require.NoError(repo.SubscribeChannel(user.ID, ch.ID))
		}
		for i := 0; i < 10; i++ {
			mustMakeMessage(t, repo, user.ID, ch.ID)
		}
		latests = append(latests, mustMakeMessage(t, repo, user.ID, ch.ID).ID)
	}

	t.Run("SubTest1", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		arr, err := repo.GetChannelLatestMessagesByUserID(user.ID, -1, false)
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
		assert := assert.New(t)

		arr, err := repo.GetChannelLatestMessagesByUserID(user.ID, -1, true)
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
		assert := assert.New(t)

		arr, err := repo.GetChannelLatestMessagesByUserID(user.ID, 5, false)
		derefs := make([]uuid.UUID, len(arr))
		for i := range arr {
			derefs[i] = arr[i].ID
		}
		if assert.NoError(err) {
			assert.ElementsMatch(derefs, latests[5:])
		}
	})
}

func TestRepositoryImpl_GetArchivedMessagesByID(t *testing.T) {
	t.Parallel()
	repo, _, require, user, channel := setupWithUserAndChannel(t, common)

	cases := []string{
		"v0",
		"v1",
		"v2",
		"v3",
		"v4",
		"v5",
	}

	m, err := repo.CreateMessage(user.ID, channel.ID, cases[0])
	require.NoError(err)
	for i := 1; i < len(cases); i++ {
		require.NoError(repo.UpdateMessage(m.ID, cases[i]))
	}

	t.Run("Nil id", func(t *testing.T) {
		t.Parallel()
		assert, _ := assertAndRequire(t)

		r, err := repo.GetArchivedMessagesByID(uuid.Nil)
		if assert.NoError(err) {
			assert.Len(r, 0)
		}
	})

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		assert, _ := assertAndRequire(t)

		r, err := repo.GetArchivedMessagesByID(m.ID)
		if assert.NoError(err) && assert.Len(r, 5) {
			for i, v := range r {
				assert.Equal(cases[i], v.Text)
			}
		}
	})
}
