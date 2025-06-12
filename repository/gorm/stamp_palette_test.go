package gorm

import (
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/utils/optional"
	random2 "github.com/traPtitech/traQ/utils/random"
)

func TestRepositoryImpl_CreateStampPalette(t *testing.T) {
	t.Parallel()
	repo, _, _, user := setupWithUser(t, common2)

	t.Run("nil user id", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		_, err := repo.CreateStampPalette(random2.AlphaNumeric(20), random2.AlphaNumeric(100), make([]uuid.UUID, 0), uuid.Nil)
		assert.Error(err)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		name := random2.AlphaNumeric(20)
		description := random2.AlphaNumeric(100)
		stamps := make([]uuid.UUID, 0)
		n := 100
		for range n {
			s := mustMakeStamp(t, repo, rand, user.GetID())
			stamps = append(stamps, s.ID)
		}
		sp, err := repo.CreateStampPalette(name, description, stamps, user.GetID())
		if assert.NoError(err) {
			assert.NotEmpty(sp.ID)
			assert.Equal(name, sp.Name)
			assert.Equal(description, sp.Description)
			assert.Equal(user.GetID(), sp.CreatorID)
			assert.EqualValues(stamps, sp.Stamps)
			assert.NotEmpty(sp.CreatedAt)
			assert.NotEmpty(sp.UpdatedAt)
		}
	})
}

func TestRepositoryImpl_UpdateStampPalette(t *testing.T) {
	t.Parallel()
	repo, _, _, user := setupWithUser(t, common2)

	stampPalette := mustMakeStampPalette(t, repo, rand, rand, make([]uuid.UUID, 0), user.GetID())

	t.Run("nil id", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		assert.EqualError(repo.UpdateStampPalette(uuid.Nil, repository.UpdateStampPaletteArgs{}), repository.ErrNilID.Error())
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		assert.EqualError(repo.UpdateStampPalette(uuid.Must(uuid.NewV7()), repository.UpdateStampPaletteArgs{}), repository.ErrNotFound.Error())
	})

	t.Run("no change", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		assert.NoError(repo.UpdateStampPalette(stampPalette.ID, repository.UpdateStampPaletteArgs{}))
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		assert, require := assertAndRequire(t)

		stampPalette := mustMakeStampPalette(t, repo, rand, rand, make([]uuid.UUID, 0), user.GetID())
		newName := random2.AlphaNumeric(20)
		newDescription := random2.AlphaNumeric(100)

		if assert.NoError(repo.UpdateStampPalette(stampPalette.ID, repository.UpdateStampPaletteArgs{
			Name:        optional.From(newName),
			Description: optional.From(newDescription),
			Stamps:      make([]uuid.UUID, 0),
		})) {
			newStampPalette, err := repo.GetStampPalette(stampPalette.ID)
			require.NoError(err)
			assert.Equal(newDescription, newStampPalette.Description)
			assert.Equal(newName, newStampPalette.Name)
		}
	})
}

func TestRepositoryImpl_GetStampPalette(t *testing.T) {
	t.Parallel()
	repo, _, _, user := setupWithUser(t, common2)

	t.Run("nil id", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		_, err := repo.GetStampPalette(uuid.Nil)
		assert.EqualError(err, repository.ErrNotFound.Error())
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		_, err := repo.GetStampPalette(uuid.Must(uuid.NewV7()))
		assert.EqualError(err, repository.ErrNotFound.Error())
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		createdStampPalette := mustMakeStampPalette(t, repo, rand, rand, make([]uuid.UUID, 0), user.GetID())

		stampPalette, err := repo.GetStampPalette(createdStampPalette.ID)
		if assert.NoError(err) {
			assert.Equal(createdStampPalette.ID, stampPalette.ID)
			assert.Equal(createdStampPalette.Name, stampPalette.Name)
			assert.Equal(createdStampPalette.Description, stampPalette.Description)
			assert.Equal(createdStampPalette.CreatorID, stampPalette.CreatorID)
		}
	})
}

func TestRepositoryImpl_DeleteStampPalette(t *testing.T) {
	t.Parallel()
	repo, _, _, user := setupWithUser(t, common2)

	t.Run("nil id", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		assert.EqualError(repo.DeleteStampPalette(uuid.Nil), repository.ErrNilID.Error())
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		assert.EqualError(repo.DeleteStampPalette(uuid.Must(uuid.NewV7())), repository.ErrNotFound.Error())
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		stampPalette := mustMakeStampPalette(t, repo, rand, rand, make([]uuid.UUID, 0), user.GetID())
		if assert.NoError(repo.DeleteStampPalette(stampPalette.ID)) {
			_, err := repo.GetStampPalette(stampPalette.ID)
			assert.EqualError(err, repository.ErrNotFound.Error())
		}
	})
}

func TestRepositoryImpl_GetStampPalettes(t *testing.T) {
	t.Parallel()
	repo, assert, _, user := setupWithUser(t, common2)
	otherUser := mustMakeUser(t, repo, rand)

	n := 10
	for range 10 {
		mustMakeStampPalette(t, repo, rand, rand, make([]uuid.UUID, 0), user.GetID())
	}
	mustMakeStampPalette(t, repo, rand, rand, make([]uuid.UUID, 0), otherUser.GetID())

	arr, err := repo.GetStampPalettes(user.GetID())
	if assert.NoError(err) {
		assert.Len(arr, n)
		for _, stampPalette := range arr {
			assert.Equal(user.GetID(), stampPalette.CreatorID)
		}
	}
}
