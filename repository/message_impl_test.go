package repository

import (
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/traPtitech/traQ/model"
	"testing"
)

func TestRepositoryImpl_CreateMessage(t *testing.T) {
	t.Parallel()
	repo, _, _, user, channel := setupWithUserAndChannel(t, common3)

	t.Run("failures 1", func(t *testing.T) {
		t.Parallel()

		_, err := repo.CreateMessage(user.GetID(), uuid.Nil, "a")
		assert.Error(t, err)
	})

	t.Run("failures 2", func(t *testing.T) {
		t.Parallel()

		_, err := repo.CreateMessage(uuid.Nil, channel.ID, "a")
		assert.Error(t, err)
	})

	t.Run("success 1", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		m, err := repo.CreateMessage(user.GetID(), channel.ID, "test")
		if assert.NoError(err) {
			assert.NotZero(m.ID)
			assert.Equal(user.GetID(), m.UserID)
			assert.Equal(channel.ID, m.ChannelID)
			assert.Equal("test", m.Text)
			assert.NotZero(m.CreatedAt)
			assert.NotZero(m.UpdatedAt)
			assert.Nil(m.DeletedAt)
		}

		m, err = repo.CreateMessage(user.GetID(), channel.ID, "")
		if assert.NoError(err) {
			assert.NotZero(m.ID)
			assert.Equal(user.GetID(), m.UserID)
			assert.Equal(channel.ID, m.ChannelID)
			assert.Equal("", m.Text)
			assert.NotZero(m.CreatedAt)
			assert.NotZero(m.UpdatedAt)
			assert.Nil(m.DeletedAt)
		}
	})

	t.Run("success 2", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		m, err := repo.CreateMessage(user.GetID(), channel.ID, "")
		if assert.NoError(err) {
			assert.NotZero(m.ID)
			assert.Equal(user.GetID(), m.UserID)
			assert.Equal(channel.ID, m.ChannelID)
			assert.Equal("", m.Text)
			assert.NotZero(m.CreatedAt)
			assert.NotZero(m.UpdatedAt)
			assert.Nil(m.DeletedAt)
		}
	})
}

func TestRepositoryImpl_UpdateMessage(t *testing.T) {
	t.Parallel()
	repo, assert, _, user, channel := setupWithUserAndChannel(t, common3)

	m := mustMakeMessage(t, repo, user.GetID(), channel.ID)
	originalText := m.Text

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
	repo, assert, _, user, channel := setupWithUserAndChannel(t, common3)

	m := mustMakeMessage(t, repo, user.GetID(), channel.ID)

	assert.EqualError(repo.DeleteMessage(uuid.Nil), ErrNilID.Error())

	if assert.NoError(repo.DeleteMessage(m.ID)) {
		_, err := repo.GetMessageByID(m.ID)
		assert.EqualError(err, ErrNotFound.Error())
	}
	assert.EqualError(repo.DeleteMessage(m.ID), ErrNotFound.Error())
}

func TestRepositoryImpl_GetMessageByID(t *testing.T) {
	t.Parallel()
	repo, assert, _, user, channel := setupWithUserAndChannel(t, common3)

	m := mustMakeMessage(t, repo, user.GetID(), channel.ID)

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
	repo, assert, _, user, channel := setupWithUserAndChannel(t, common3)

	m := mustMakeMessage(t, repo, user.GetID(), channel.ID)

	assert.Error(repo.SetMessageUnread(uuid.Nil, m.ID, true))
	assert.Error(repo.SetMessageUnread(user.GetID(), uuid.Nil, true))
	if assert.NoError(repo.SetMessageUnread(user.GetID(), m.ID, true)) {
		assert.Equal(1, count(t, getDB(repo).Model(model.Unread{}).Where(model.Unread{UserID: user.GetID()})))
	}
	assert.NoError(repo.SetMessageUnread(user.GetID(), m.ID, true))
}

func TestRepositoryImpl_GetUnreadMessagesByUserID(t *testing.T) {
	t.Parallel()
	repo, assert, _, user, channel := setupWithUserAndChannel(t, common3)

	for i := 0; i < 10; i++ {
		mustMakeMessageUnread(t, repo, user.GetID(), mustMakeMessage(t, repo, user.GetID(), channel.ID).ID)
	}

	if unreads, err := repo.GetUnreadMessagesByUserID(user.GetID()); assert.NoError(err) {
		assert.Len(unreads, 10)
	}
	if unreads, err := repo.GetUnreadMessagesByUserID(uuid.Nil); assert.NoError(err) {
		assert.Len(unreads, 0)
	}
}

func TestRepositoryImpl_DeleteUnreadsByChannelID(t *testing.T) {
	t.Parallel()
	repo, _, _, user, channel := setupWithUserAndChannel(t, common3)

	creator := mustMakeUser(t, repo, rand)
	channel2 := mustMakeChannel(t, repo, rand)
	testMessage := mustMakeMessage(t, repo, creator.GetID(), channel.ID)
	testMessage2 := mustMakeMessage(t, repo, creator.GetID(), channel2.ID)
	mustMakeMessageUnread(t, repo, user.GetID(), testMessage.ID)
	mustMakeMessageUnread(t, repo, user.GetID(), testMessage2.ID)

	t.Run("nil id", func(t *testing.T) {
		t.Parallel()

		assert.EqualError(t, repo.DeleteUnreadsByChannelID(channel.ID, uuid.Nil), ErrNilID.Error())
		assert.EqualError(t, repo.DeleteUnreadsByChannelID(uuid.Nil, user.GetID()), ErrNilID.Error())
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		if assert.NoError(repo.DeleteUnreadsByChannelID(channel.ID, user.GetID())) {
			assert.Equal(1, count(t, getDB(repo).Model(model.Unread{}).Where(&model.Unread{UserID: user.GetID()})))
		}
	})
}

func TestRepositoryImpl_GetChannelLatestMessagesByUserID(t *testing.T) {
	t.Parallel()
	repo, _, _, user := setupWithUser(t, ex1)

	var latests []uuid.UUID
	for j := 0; j < 10; j++ {
		ch := mustMakeChannel(t, repo, rand)
		if j < 5 {
			mustChangeChannelSubscription(t, repo, ch.ID, user.GetID())
		}
		for i := 0; i < 10; i++ {
			mustMakeMessage(t, repo, user.GetID(), ch.ID)
		}
		latests = append(latests, mustMakeMessage(t, repo, user.GetID(), ch.ID).ID)
	}

	t.Run("SubTest1", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		arr, err := repo.GetChannelLatestMessagesByUserID(user.GetID(), -1, false)
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

		arr, err := repo.GetChannelLatestMessagesByUserID(user.GetID(), -1, true)
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

		arr, err := repo.GetChannelLatestMessagesByUserID(user.GetID(), 5, false)
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
	repo, _, require, user, channel := setupWithUserAndChannel(t, common3)

	cases := []string{
		"v0",
		"v1",
		"v2",
		"v3",
		"v4",
		"v5",
	}

	m, err := repo.CreateMessage(user.GetID(), channel.ID, cases[0])
	require.NoError(err)
	for i := 1; i < len(cases); i++ {
		require.NoError(repo.UpdateMessage(m.ID, cases[i]))
	}

	t.Run("Nil id", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		r, err := repo.GetArchivedMessagesByID(uuid.Nil)
		if assert.NoError(err) {
			assert.Len(r, 0)
		}
	})

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		r, err := repo.GetArchivedMessagesByID(m.ID)
		if assert.NoError(err) && assert.Len(r, 5) {
			for i, v := range r {
				assert.Equal(cases[i], v.Text)
			}
		}
	})
}

func TestRepositoryImpl_AddStampToMessage(t *testing.T) {
	t.Parallel()
	repo, _, _, user, channel := setupWithUserAndChannel(t, common3)

	message := mustMakeMessage(t, repo, user.GetID(), channel.ID)
	stamp := mustMakeStamp(t, repo, rand, uuid.Nil)

	t.Run("Nil id", func(t *testing.T) {
		t.Parallel()

		_, err := repo.AddStampToMessage(uuid.Nil, uuid.Nil, uuid.Nil, 1)
		assert.EqualError(t, err, ErrNilID.Error())
	})

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		{
			ms, err := repo.AddStampToMessage(message.ID, stamp.ID, user.GetID(), 1)
			if assert.NoError(err) {
				assert.Equal(message.ID, ms.MessageID)
				assert.Equal(stamp.ID, ms.StampID)
				assert.Equal(user.GetID(), ms.UserID)
				assert.Equal(1, ms.Count)
				assert.NotEmpty(ms.CreatedAt)
				assert.NotEmpty(ms.UpdatedAt)
			}
		}
		{
			ms, err := repo.AddStampToMessage(message.ID, stamp.ID, user.GetID(), 1)
			if assert.NoError(err) {
				assert.Equal(message.ID, ms.MessageID)
				assert.Equal(stamp.ID, ms.StampID)
				assert.Equal(user.GetID(), ms.UserID)
				assert.Equal(2, ms.Count)
				assert.NotEmpty(ms.CreatedAt)
				assert.NotEmpty(ms.UpdatedAt)
			}
		}
		{
			ms, err := repo.AddStampToMessage(message.ID, stamp.ID, user.GetID(), 3)
			if assert.NoError(err) {
				assert.Equal(message.ID, ms.MessageID)
				assert.Equal(stamp.ID, ms.StampID)
				assert.Equal(user.GetID(), ms.UserID)
				assert.Equal(5, ms.Count)
				assert.NotEmpty(ms.CreatedAt)
				assert.NotEmpty(ms.UpdatedAt)
			}
		}
	})
}

func TestRepositoryImpl_RemoveStampFromMessage(t *testing.T) {
	t.Parallel()
	repo, _, _, user, channel := setupWithUserAndChannel(t, common3)

	message := mustMakeMessage(t, repo, user.GetID(), channel.ID)
	stamp := mustMakeStamp(t, repo, rand, uuid.Nil)

	t.Run("Nil id", func(t *testing.T) {
		t.Parallel()
		assert.EqualError(t, repo.RemoveStampFromMessage(message.ID, stamp.ID, uuid.Nil), ErrNilID.Error())
		assert.EqualError(t, repo.RemoveStampFromMessage(message.ID, uuid.Nil, user.GetID()), ErrNilID.Error())
		assert.EqualError(t, repo.RemoveStampFromMessage(uuid.Nil, stamp.ID, user.GetID()), ErrNilID.Error())
	})

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		mustAddMessageStamp(t, repo, message.ID, stamp.ID, user.GetID())
		mustAddMessageStamp(t, repo, message.ID, stamp.ID, user.GetID())

		if assert.NoError(t, repo.RemoveStampFromMessage(message.ID, stamp.ID, user.GetID())) {
			assert.Equal(t, 0, count(t, getDB(repo).Model(&model.MessageStamp{}).Where(&model.MessageStamp{MessageID: message.ID, StampID: stamp.ID, UserID: user.GetID()})))
		}
	})
}
