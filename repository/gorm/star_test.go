package gorm

import (
	"testing"

	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/model"
)

func TestRepositoryImpl_AddStar(t *testing.T) {
	t.Parallel()
	repo, assert, _, user, channel := setupWithUserAndChannel(t, common2)

	assert.Error(repo.AddStar(user.GetID(), uuid.Nil))
	assert.Error(repo.AddStar(uuid.Nil, channel.ID))
	if assert.NoError(repo.AddStar(user.GetID(), channel.ID)) {
		assert.Equal(1, count(t, getDB(repo).Model(model.Star{}).Where(model.Star{UserID: user.GetID()})))
	}
	if assert.NoError(repo.AddStar(user.GetID(), channel.ID)) {
		assert.Equal(1, count(t, getDB(repo).Model(model.Star{}).Where(model.Star{UserID: user.GetID()})))
	}
}

func TestRepositoryImpl_RemoveStar(t *testing.T) {
	t.Parallel()
	repo, assert, require, user, channel := setupWithUserAndChannel(t, common2)

	require.NoError(repo.AddStar(user.GetID(), channel.ID))

	assert.Error(repo.RemoveStar(uuid.Nil, channel.ID))
	assert.Error(repo.RemoveStar(user.GetID(), uuid.Nil))

	if assert.NoError(repo.RemoveStar(user.GetID(), channel.ID)) {
		assert.Equal(0, count(t, getDB(repo).Model(model.Star{}).Where(model.Star{UserID: user.GetID(), ChannelID: channel.ID})))
	}
	if assert.NoError(repo.RemoveStar(user.GetID(), channel.ID)) {
		assert.Equal(0, count(t, getDB(repo).Model(model.Star{}).Where(model.Star{UserID: user.GetID(), ChannelID: channel.ID})))
	}
}

func TestRepositoryImpl_GetStaredChannels(t *testing.T) {
	t.Parallel()
	repo, assert, require, user, _ := setupWithUserAndChannel(t, common2)

	n := 5
	for _ = range n {
		ch := mustMakeChannel(t, repo, rand)
		require.NoError(repo.AddStar(user.GetID(), ch.ID))
	}

	ch, err := repo.GetStaredChannels(user.GetID())
	if assert.NoError(err) {
		assert.Len(ch, n)
	}

	ch, err = repo.GetStaredChannels(uuid.Nil)
	if assert.NoError(err) {
		assert.Len(ch, 0)
	}
}
