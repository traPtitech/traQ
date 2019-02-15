package impl

import (
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/model"
	"testing"
)

func TestRepositoryImpl_AddStar(t *testing.T) {
	t.Parallel()
	repo, assert, _, user, channel := setupWithUserAndChannel(t, common)

	assert.Error(repo.AddStar(user.ID, uuid.Nil))
	assert.Error(repo.AddStar(uuid.Nil, channel.ID))
	if assert.NoError(repo.AddStar(user.ID, channel.ID)) {
		assert.Equal(1, count(t, getDB(repo).Model(model.Star{}).Where(model.Star{UserID: user.ID})))
	}
	if assert.NoError(repo.AddStar(user.ID, channel.ID)) {
		assert.Equal(1, count(t, getDB(repo).Model(model.Star{}).Where(model.Star{UserID: user.ID})))
	}
}

func TestRepositoryImpl_RemoveStar(t *testing.T) {
	t.Parallel()
	repo, assert, require, user, channel := setupWithUserAndChannel(t, common)

	require.NoError(repo.AddStar(user.ID, channel.ID))

	assert.Error(repo.RemoveStar(uuid.Nil, channel.ID))
	assert.Error(repo.RemoveStar(user.ID, uuid.Nil))

	if assert.NoError(repo.RemoveStar(user.ID, channel.ID)) {
		assert.Equal(0, count(t, getDB(repo).Model(model.Star{}).Where(model.Star{UserID: user.ID, ChannelID: channel.ID})))
	}
	if assert.NoError(repo.RemoveStar(user.ID, channel.ID)) {
		assert.Equal(0, count(t, getDB(repo).Model(model.Star{}).Where(model.Star{UserID: user.ID, ChannelID: channel.ID})))
	}
}

func TestRepositoryImpl_GetStaredChannels(t *testing.T) {
	t.Parallel()
	repo, assert, require, user, _ := setupWithUserAndChannel(t, common)

	n := 5
	for i := 0; i < n; i++ {
		ch := mustMakeChannel(t, repo, random)
		require.NoError(repo.AddStar(user.ID, ch.ID))
	}

	ch, err := repo.GetStaredChannels(user.ID)
	if assert.NoError(err) {
		assert.Len(ch, n)
	}

	ch, err = repo.GetStaredChannels(uuid.Nil)
	if assert.NoError(err) {
		assert.Len(ch, 0)
	}
}
