package repository

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/utils"
	"gopkg.in/guregu/null.v3"
	"testing"
)

func TestRepositoryImpl_CreateStampPalette(t *testing.T) {
	t.Parallel()
	repo, _, _, user := setupWithUser(t, common)

	t.Run("nil user id", func(t *testing.T) {
		t.Parallel()
		assert, _ := assertAndRequire(t)

		_, err := repo.CreateStampPalette(utils.RandAlphabetAndNumberString(20), utils.RandAlphabetAndNumberString(100), make([]uuid.UUID, 0), uuid.Nil)
		assert.Error(err)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		assert, _ := assertAndRequire(t)

		name := utils.RandAlphabetAndNumberString(20)
		description := utils.RandAlphabetAndNumberString(100)
		stamps := make([]uuid.UUID, 0)
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
	repo, _, _, user := setupWithUser(t, common)

	stampPalette := mustMakeStampPalette(t, repo, random, random, make([]uuid.UUID, 0), user.GetID())

	t.Run("nil id", func(t *testing.T) {
		t.Parallel()
		assert, _ := assertAndRequire(t)

		assert.EqualError(repo.UpdateStampPalette(uuid.Nil, UpdateStampPaletteArgs{}), ErrNilID.Error())
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		assert, _ := assertAndRequire(t)

		assert.EqualError(repo.UpdateStampPalette(uuid.Must(uuid.NewV4()), UpdateStampPaletteArgs{}), ErrNotFound.Error())
	})

	t.Run("no change", func(t *testing.T) {
		t.Parallel()
		assert, _ := assertAndRequire(t)

		assert.NoError(repo.UpdateStampPalette(stampPalette.ID, UpdateStampPaletteArgs{}))
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		assert, require := assertAndRequire(t)

		stampPalette := mustMakeStampPalette(t, repo, random, random, make([]uuid.UUID, 0), user.GetID())
		newName := utils.RandAlphabetAndNumberString(20)
		newDescription := utils.RandAlphabetAndNumberString(100)

		if assert.NoError(repo.UpdateStampPalette(stampPalette.ID, UpdateStampPaletteArgs{
			Name:        null.StringFrom(newName),
			Description: null.StringFrom(newDescription),
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
	repo, _, _, user := setupWithUser(t, common)

	t.Run("nil id", func(t *testing.T) {
		t.Parallel()
		assert, _ := assertAndRequire(t)

		_, err := repo.GetStampPalette(uuid.Nil)
		assert.EqualError(err, ErrNotFound.Error())
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		assert, _ := assertAndRequire(t)

		_, err := repo.GetStampPalette(uuid.Must(uuid.NewV4()))
		assert.EqualError(err, ErrNotFound.Error())
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		assert, _ := assertAndRequire(t)
		createdStampPalette := mustMakeStampPalette(t, repo, random, random, make([]uuid.UUID, 0), user.GetID())

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
	repo, _, _, user := setupWithUser(t, common)

	t.Run("nil id", func(t *testing.T) {
		t.Parallel()
		assert, _ := assertAndRequire(t)

		assert.EqualError(repo.DeleteStampPalette(uuid.Nil), ErrNilID.Error())
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		assert, _ := assertAndRequire(t)

		assert.EqualError(repo.DeleteStampPalette(uuid.Must(uuid.NewV4())), ErrNotFound.Error())
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		assert, _ := assertAndRequire(t)

		stampPalette := mustMakeStampPalette(t, repo, random, random, make([]uuid.UUID, 0), user.GetID())
		if assert.NoError(repo.DeleteStampPalette(stampPalette.ID)) {
			_, err := repo.GetStampPalette(stampPalette.ID)
			assert.EqualError(err, ErrNotFound.Error())
		}
	})
}

func TestRepositoryImpl_GetStampPalettes(t *testing.T) {
	t.Parallel()
	repo, assert, _, user := setupWithUser(t, common)
	otherUser := mustMakeUser(t, repo, random)

	n := 10
	for i := 0; i < 10; i++ {
		mustMakeStampPalette(t, repo, random, random, make([]uuid.UUID, 0), user.GetID())
	}
	mustMakeStampPalette(t, repo, random, random, make([]uuid.UUID, 0), otherUser.GetID())

	arr, err := repo.GetStampPalettes(user.GetID())
	if assert.NoError(err) {
		assert.Len(arr, n)
		for _, stampPalette := range arr {
			assert.Equal(user.GetID(), stampPalette.CreatorID)
		}
	}
}
