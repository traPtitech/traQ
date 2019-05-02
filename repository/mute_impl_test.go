package repository

import (
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/traPtitech/traQ/model"
	"testing"
)

func TestRepositoryImpl_MuteChannel(t *testing.T) {
	t.Parallel()
	repo, assert, _, user, channel := setupWithUserAndChannel(t, common)

	assert.EqualError(repo.MuteChannel(user.ID, uuid.Nil), ErrNilID.Error())
	assert.EqualError(repo.MuteChannel(uuid.Nil, channel.ID), ErrNilID.Error())
	if assert.NoError(repo.MuteChannel(user.ID, channel.ID)) {
		assert.Equal(1, count(t, getDB(repo).Model(model.Mute{}).Where(model.Mute{UserID: user.ID})))
	}
	if assert.NoError(repo.MuteChannel(user.ID, channel.ID)) {
		assert.Equal(1, count(t, getDB(repo).Model(model.Mute{}).Where(model.Mute{UserID: user.ID})))
	}
}

func TestRepositoryImpl_UnmuteChannel(t *testing.T) {
	t.Parallel()
	repo, assert, require, user, channel := setupWithUserAndChannel(t, common)

	require.NoError(repo.MuteChannel(user.ID, channel.ID))

	assert.EqualError(repo.UnmuteChannel(uuid.Nil, channel.ID), ErrNilID.Error())
	assert.EqualError(repo.UnmuteChannel(user.ID, uuid.Nil), ErrNilID.Error())

	if assert.NoError(repo.UnmuteChannel(user.ID, channel.ID)) {
		assert.Equal(0, count(t, getDB(repo).Model(model.Mute{}).Where(model.Mute{UserID: user.ID, ChannelID: channel.ID})))
	}
	if assert.NoError(repo.UnmuteChannel(user.ID, channel.ID)) {
		assert.Equal(0, count(t, getDB(repo).Model(model.Mute{}).Where(model.Mute{UserID: user.ID, ChannelID: channel.ID})))
	}
}

func TestRepositoryImpl_GetMutedChannelIDs(t *testing.T) {
	t.Parallel()
	repo, _, require, user, channel := setupWithUserAndChannel(t, common)

	channel2 := mustMakeChannel(t, repo, random)
	user2 := mustMakeUser(t, repo, random)

	require.NoError(repo.MuteChannel(user.ID, channel.ID))
	require.NoError(repo.MuteChannel(user2.ID, channel.ID))
	require.NoError(repo.MuteChannel(user.ID, channel2.ID))

	t.Run("nil id", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		ids, err := repo.GetMutedChannelIDs(uuid.Nil)
		if assert.NoError(err) {
			assert.Empty(ids)
		}
	})

	t.Run("success1", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		ids, err := repo.GetMutedChannelIDs(user.ID)
		if assert.NoError(err) {
			assert.ElementsMatch(ids, []uuid.UUID{channel.ID, channel2.ID})
		}
	})

	t.Run("success2", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		ids, err := repo.GetMutedChannelIDs(user2.ID)
		if assert.NoError(err) {
			assert.ElementsMatch(ids, []uuid.UUID{channel.ID})
		}
	})
}

func TestRepositoryImpl_GetMuteUserIDs(t *testing.T) {
	t.Parallel()
	repo, _, require, user, channel := setupWithUserAndChannel(t, common)

	channel2 := mustMakeChannel(t, repo, random)
	user2 := mustMakeUser(t, repo, random)

	require.NoError(repo.MuteChannel(user.ID, channel.ID))
	require.NoError(repo.MuteChannel(user2.ID, channel.ID))
	require.NoError(repo.MuteChannel(user.ID, channel2.ID))

	t.Run("nil id", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		ids, err := repo.GetMuteUserIDs(uuid.Nil)
		if assert.NoError(err) {
			assert.Empty(ids)
		}
	})

	t.Run("success1", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		ids, err := repo.GetMuteUserIDs(channel.ID)
		if assert.NoError(err) {
			assert.ElementsMatch(ids, []uuid.UUID{user.ID, user2.ID})
		}
	})

	t.Run("success2", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		ids, err := repo.GetMuteUserIDs(channel2.ID)
		if assert.NoError(err) {
			assert.ElementsMatch(ids, []uuid.UUID{user.ID})
		}
	})
}

func TestRepositoryImpl_IsChannelMuted(t *testing.T) {
	t.Parallel()
	repo, _, require, user, channel := setupWithUserAndChannel(t, common)

	user2 := mustMakeUser(t, repo, random)
	require.NoError(repo.MuteChannel(user.ID, channel.ID))

	t.Run("nil id", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		ok, err := repo.IsChannelMuted(uuid.Nil, channel.ID)
		if assert.NoError(err) {
			assert.False(ok)
		}
	})

	t.Run("success1", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		ok, err := repo.IsChannelMuted(user.ID, channel.ID)
		if assert.NoError(err) {
			assert.True(ok)
		}
	})

	t.Run("success2", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		ok, err := repo.IsChannelMuted(user2.ID, channel.ID)
		if assert.NoError(err) {
			assert.False(ok)
		}
	})
}
