package gorm

import (
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/traPtitech/traQ/repository"
)

func TestRepositoryImpl_PinMessage(t *testing.T) {
	t.Parallel()
	repo, _, _, user, channel := setupWithUserAndChannel(t, common2)

	t.Run("nil id (message)", func(t *testing.T) {
		t.Parallel()

		_, err := repo.PinMessage(uuid.Nil, user.GetID())
		assert.Error(t, err)
	})

	t.Run("nil id (user)", func(t *testing.T) {
		t.Parallel()

		_, err := repo.PinMessage(uuid.Must(uuid.NewV7()), uuid.Nil)
		assert.Error(t, err)
	})

	t.Run("message not found", func(t *testing.T) {
		t.Parallel()

		_, err := repo.PinMessage(uuid.Must(uuid.NewV7()), user.GetID())
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		testMessage := mustMakeMessage(t, repo, user.GetID(), channel.ID)

		p, err := repo.PinMessage(testMessage.ID, user.GetID())
		if assert.NoError(t, err) {
			assert.EqualValues(t, testMessage.ID, p.MessageID)
		}
	})

	t.Run("duplicate", func(t *testing.T) {
		t.Parallel()
		testMessage := mustMakeMessage(t, repo, user.GetID(), channel.ID)
		mustMakePin(t, repo, testMessage.ID, user.GetID())

		_, err := repo.PinMessage(testMessage.ID, user.GetID())
		assert.EqualError(t, err, repository.ErrAlreadyExists.Error())
	})
}

func TestRepositoryImpl_UnpinMessage(t *testing.T) {
	t.Parallel()
	repo, _, _, user, channel := setupWithUserAndChannel(t, common2)

	t.Run("nil id", func(t *testing.T) {
		t.Parallel()

		_, err := repo.UnpinMessage(uuid.Nil)
		assert.Error(t, err)
	})

	t.Run("pin not found", func(t *testing.T) {
		t.Parallel()

		_, err := repo.PinMessage(uuid.Must(uuid.NewV7()), user.GetID())
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		testMessage := mustMakeMessage(t, repo, user.GetID(), channel.ID)
		mustMakePin(t, repo, testMessage.ID, user.GetID())

		pin, err := repo.UnpinMessage(testMessage.ID)
		if assert.NoError(t, err) {
			assert.EqualValues(t, user.GetID(), pin.UserID)
			assert.EqualValues(t, testMessage.ID, pin.MessageID)
		}
	})
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
