package repository

import (
	"github.com/gofrs/uuid"
	"testing"
)

func TestRepositoryImpl_PinMessage(t *testing.T) {
	t.Parallel()
	repo, assert, _, user, channel := setupWithUserAndChannel(t, common2)

	testMessage := mustMakeMessage(t, repo, user.GetID(), channel.ID)

	_, err := repo.PinMessage(uuid.Nil, user.GetID())
	assert.Error(err)

	_, err = repo.PinMessage(testMessage.ID, uuid.Nil)
	assert.Error(err)

	p, err := repo.PinMessage(testMessage.ID, user.GetID())
	if assert.NoError(err) {
		assert.NotEmpty(p.ID)
	}

	p2, err := repo.PinMessage(testMessage.ID, user.GetID())
	if assert.NoError(err) {
		assert.EqualValues(p.ID, p2.ID)
	}
}

func TestRepositoryImpl_UnpinMessage(t *testing.T) {
	t.Parallel()
	repo, assert, _, user, channel := setupWithUserAndChannel(t, common2)

	testMessage := mustMakeMessage(t, repo, user.GetID(), channel.ID)
	mustMakePin(t, repo, testMessage.ID, user.GetID())

	assert.Error(repo.UnpinMessage(uuid.Nil, user.GetID()))
	assert.NoError(repo.UnpinMessage(testMessage.ID, user.GetID()))
	assert.NoError(repo.UnpinMessage(uuid.Must(uuid.NewV4()), user.GetID()))
}

func TestRepositoryImpl_GetPinnedMessageByChannelID(t *testing.T) {
	t.Parallel()
	repo, assert, _, user, channel := setupWithUserAndChannel(t, common2)

	testMessage := mustMakeMessage(t, repo, user.GetID(), channel.ID)
	mustMakePin(t, repo, testMessage.ID, user.GetID())

	pins, err := repo.GetPinnedMessageByChannelID(channel.ID)
	if assert.NoError(err) {
		assert.Len(pins, 1)
	}

	pins, err = repo.GetPinnedMessageByChannelID(uuid.Nil)
	if assert.NoError(err) {
		assert.Empty(pins)
	}
}
