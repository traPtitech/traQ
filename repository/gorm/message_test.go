package gorm

import (
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/utils/optional"
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
			assert.False(m.DeletedAt.Valid)
		}

		m, err = repo.CreateMessage(user.GetID(), channel.ID, "")
		if assert.NoError(err) {
			assert.NotZero(m.ID)
			assert.Equal(user.GetID(), m.UserID)
			assert.Equal(channel.ID, m.ChannelID)
			assert.Equal("", m.Text)
			assert.NotZero(m.CreatedAt)
			assert.NotZero(m.UpdatedAt)
			assert.False(m.DeletedAt.Valid)
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
			assert.False(m.DeletedAt.Valid)
		}
	})
}

func TestRepositoryImpl_UpdateMessage(t *testing.T) {
	t.Parallel()
	repo, assert, _, user, channel := setupWithUserAndChannel(t, common3)

	m := mustMakeMessage(t, repo, user.GetID(), channel.ID)
	originalText := m.Text

	assert.EqualError(repo.UpdateMessage(uuid.Must(uuid.NewV4()), "new message"), repository.ErrNotFound.Error())
	assert.EqualError(repo.UpdateMessage(uuid.Must(uuid.NewV7()), "new message"), repository.ErrNotFound.Error())
	assert.EqualError(repo.UpdateMessage(uuid.Nil, "new message"), repository.ErrNilID.Error())
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

	assert.EqualError(repo.DeleteMessage(uuid.Nil), repository.ErrNilID.Error())

	if assert.NoError(repo.DeleteMessage(m.ID)) {
		_, err := repo.GetMessageByID(m.ID)
		assert.EqualError(err, repository.ErrNotFound.Error())
	}
	assert.EqualError(repo.DeleteMessage(m.ID), repository.ErrNotFound.Error())
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

	_, err = repo.GetMessageByID(uuid.Must(uuid.NewV7()))
	assert.Error(err)
}

func TestRepositoryImpl_GetMessages(t *testing.T) {
	t.Parallel()
	repo, _, require, user := setupWithUser(t, ex3)
	ch1 := mustMakeChannel(t, repo, rand)
	ch2 := mustMakeChannel(t, repo, rand)

	_, _, err := repo.ChangeChannelSubscription(ch1.ID, repository.ChangeChannelSubscriptionArgs{
		Subscription: map[uuid.UUID]model.ChannelSubscribeLevel{
			user.GetID(): model.ChannelSubscribeLevelMarkAndNotify,
		},
	})
	require.NoError(err)

	for range 5 {
		mustMakeMessage(t, repo, user.GetID(), ch1.ID)
	}
	m6 := mustMakeMessage(t, repo, user.GetID(), ch1.ID)
	latestCh1Msg := mustMakeMessage(t, repo, user.GetID(), ch1.ID)
	latestCh2Msg := mustMakeMessage(t, repo, user.GetID(), ch2.ID)

	messageEquals := func(t *testing.T, expected, actual *model.Message) {
		t.Helper()

		assert.EqualValues(t, expected.ID, actual.ID)
		assert.EqualValues(t, expected.Text, actual.Text)
		assert.EqualValues(t, expected.UserID, actual.UserID)
		assert.EqualValues(t, expected.ChannelID, actual.ChannelID)
		assert.NotEmpty(t, actual.CreatedAt)
		assert.NotEmpty(t, actual.UpdatedAt)
	}

	t.Run("id in", func(t *testing.T) {
		t.Parallel()

		messages, more, err := repo.GetMessages(repository.MessagesQuery{
			IDIn: optional.From([]uuid.UUID{m6.ID, latestCh1Msg.ID}),
		})

		if assert.NoError(t, err) {
			assert.False(t, more)
			assert.EqualValues(t, 2, len(messages))
			messageEquals(t, latestCh1Msg, messages[0])
			messageEquals(t, m6, messages[1])
		}
	})
	t.Run("activity all", func(t *testing.T) {
		t.Parallel()

		messages, more, err := repo.GetMessages(repository.MessagesQuery{
			Since:          optional.From(time.Now().Add(-7 * 24 * time.Hour)),
			Limit:          50,
			ExcludeDMs:     true,
			DisablePreload: true,
		})

		if assert.NoError(t, err) {
			assert.False(t, more)
			assert.EqualValues(t, 8, len(messages))
			messageEquals(t, latestCh2Msg, messages[0])
			messageEquals(t, latestCh1Msg, messages[1])
		}
	})

	t.Run("activity all with limit", func(t *testing.T) {
		t.Parallel()

		messages, more, err := repo.GetMessages(repository.MessagesQuery{
			Since:          optional.From(time.Now().Add(-7 * 24 * time.Hour)),
			Limit:          5,
			ExcludeDMs:     true,
			DisablePreload: true,
		})

		if assert.NoError(t, err) {
			assert.True(t, more)
			assert.EqualValues(t, 5, len(messages))
			messageEquals(t, latestCh2Msg, messages[0])
			messageEquals(t, latestCh1Msg, messages[1])
		}
	})

	t.Run("activity subscription", func(t *testing.T) {
		t.Parallel()

		messages, more, err := repo.GetMessages(repository.MessagesQuery{
			Since:                    optional.From(time.Now().Add(-7 * 24 * time.Hour)),
			Limit:                    50,
			ExcludeDMs:               true,
			DisablePreload:           true,
			ChannelsSubscribedByUser: user.GetID(),
		})

		if assert.NoError(t, err) {
			assert.False(t, more)
			assert.EqualValues(t, 7, len(messages))
			messageEquals(t, latestCh1Msg, messages[0])
		}
	})
}

func TestRepositoryImpl_SetMessageUnreads(t *testing.T) {
	t.Parallel()
	repo, assert, _, user1, channel := setupWithUserAndChannel(t, common3)
	user2 := mustMakeUser(t, repo, rand)
	user3 := mustMakeUser(t, repo, rand)

	m := mustMakeMessage(t, repo, user1.GetID(), channel.ID)

	assert.NoError(repo.SetMessageUnreads(nil, m.ID))
	assert.Error(repo.SetMessageUnreads(map[uuid.UUID]bool{uuid.Nil: true}, uuid.Nil))
	if assert.NoError(repo.SetMessageUnreads(map[uuid.UUID]bool{user1.GetID(): true, user2.GetID(): false}, m.ID)) {
		assert.Equal(1, count(t, getDB(repo).Model(model.Unread{}).Where(model.Unread{UserID: user1.GetID()})))
		assert.Equal(1, count(t, getDB(repo).Model(model.Unread{}).Where(model.Unread{UserID: user2.GetID()})))
		var messageCreatedAt time.Time
		assert.NoError(getDB(repo).Model(model.Unread{}).Where(model.Unread{UserID: user1.GetID()}).Select("message_created_at").Row().Scan(&messageCreatedAt))
		assert.Equal(true, m.CreatedAt.Equal(messageCreatedAt))
		assert.NoError(getDB(repo).Model(model.Unread{}).Where(model.Unread{UserID: user2.GetID()}).Select("message_created_at").Row().Scan(&messageCreatedAt))
		assert.Equal(true, m.CreatedAt.Equal(messageCreatedAt))
		var noticeable bool
		assert.NoError(getDB(repo).Model(model.Unread{}).Where(model.Unread{UserID: user1.GetID()}).Select("noticeable").Row().Scan(&noticeable))
		assert.Equal(true, noticeable)
		assert.NoError(getDB(repo).Model(model.Unread{}).Where(model.Unread{UserID: user2.GetID()}).Select("noticeable").Row().Scan(&noticeable))
		assert.Equal(false, noticeable)

	}
	if assert.NoError(repo.SetMessageUnreads(map[uuid.UUID]bool{user1.GetID(): false, user2.GetID(): true, user3.GetID(): false}, m.ID)) {
		var noticeable bool
		assert.NoError(getDB(repo).Model(model.Unread{}).Where(model.Unread{UserID: user1.GetID()}).Select("noticeable").Row().Scan(&noticeable))
		assert.Equal(false, noticeable)
		assert.NoError(getDB(repo).Model(model.Unread{}).Where(model.Unread{UserID: user2.GetID()}).Select("noticeable").Row().Scan(&noticeable))
		assert.Equal(true, noticeable)
		assert.NoError(getDB(repo).Model(model.Unread{}).Where(model.Unread{UserID: user3.GetID()}).Select("noticeable").Row().Scan(&noticeable))
		assert.Equal(false, noticeable)
	}
}

func TestRepositoryImpl_GetUnreadMessagesByUserID(t *testing.T) {
	t.Parallel()
	repo, assert, _, user, channel := setupWithUserAndChannel(t, common3)

	for range 10 {
		mustMakeMessageUnread(t, repo, user.GetID(), mustMakeMessage(t, repo, user.GetID(), channel.ID).ID)
	}

	if unreads, err := repo.GetUnreadMessagesByUserID(user.GetID()); assert.NoError(err) {
		assert.Len(unreads, 10)
	}
	if unreads, err := repo.GetUnreadMessagesByUserID(uuid.Nil); assert.NoError(err) {
		assert.Len(unreads, 0)
	}
}

func TestRepositoryImpl_GetUserUnreadChannels(t *testing.T) {
	t.Parallel()
	repo, assert, _, user, channel := setupWithUserAndChannel(t, common3)

	messages := make([]*model.Message, 10)
	for i := range 10 {
		m := mustMakeMessage(t, repo, user.GetID(), channel.ID)
		mustMakeMessageUnread(t, repo, user.GetID(), m.ID)
		messages[i] = m
	}

	t.Run("nil id", func(t *testing.T) {
		t.Parallel()

		unreads, err := repo.GetUserUnreadChannels(uuid.Nil)
		assert.NoError(err)
		assert.Len(unreads, 0)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		unreads, err := repo.GetUserUnreadChannels(user.GetID())
		assert.NoError(err)
		assert.Len(unreads, 1)

		unread := unreads[0]
		assert.Equal(channel.ID, unread.ChannelID)
		assert.Equal(10, unread.Count)
		assert.Equal(true, messages[0].CreatedAt.Equal(unread.Since))
		assert.Equal(true, messages[9].CreatedAt.Equal(unread.UpdatedAt))
		assert.Equal(messages[0].ID, unread.OldestMessageID)
	})
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

		assert.EqualError(t, repo.DeleteUnreadsByChannelID(channel.ID, uuid.Nil), repository.ErrNilID.Error())
		assert.EqualError(t, repo.DeleteUnreadsByChannelID(uuid.Nil, user.GetID()), repository.ErrNilID.Error())
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		if assert.NoError(repo.DeleteUnreadsByChannelID(channel.ID, user.GetID())) {
			assert.Equal(1, count(t, getDB(repo).Model(model.Unread{}).Where(&model.Unread{UserID: user.GetID()})))
		}
	})
}

func TestRepositoryImpl_GetChannelLatestMessages(t *testing.T) {
	t.Parallel()
	repo, _, _, user := setupWithUser(t, ex1)

	var latests []uuid.UUID
	for i := range 10 {
		ch := mustMakeChannel(t, repo, rand)
		if i < 5 {
			mustChangeChannelSubscription(t, repo, ch.ID, user.GetID())
		}
		for range 10 {
			mustMakeMessage(t, repo, user.GetID(), ch.ID)
		}
		latests = append(latests, mustMakeMessage(t, repo, user.GetID(), ch.ID).ID)
	}

	t.Run("SubTest1", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		arr, err := repo.GetChannelLatestMessages(repository.ChannelLatestMessagesQuery{})
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

		arr, err := repo.GetChannelLatestMessages(repository.ChannelLatestMessagesQuery{SubscribedByUser: optional.From(user.GetID())})
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

		arr, err := repo.GetChannelLatestMessages(repository.ChannelLatestMessagesQuery{Limit: 5})
		derefs := make([]uuid.UUID, len(arr))
		for i := range arr {
			derefs[i] = arr[i].ID
		}
		if assert.NoError(err) {
			assert.ElementsMatch(derefs, latests[5:])
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
		assert.EqualError(t, err, repository.ErrNilID.Error())
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
		assert.EqualError(t, repo.RemoveStampFromMessage(message.ID, stamp.ID, uuid.Nil), repository.ErrNilID.Error())
		assert.EqualError(t, repo.RemoveStampFromMessage(message.ID, uuid.Nil, user.GetID()), repository.ErrNilID.Error())
		assert.EqualError(t, repo.RemoveStampFromMessage(uuid.Nil, stamp.ID, user.GetID()), repository.ErrNilID.Error())
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
