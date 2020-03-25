package repository

import (
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/traPtitech/traQ/utils"
	"gopkg.in/guregu/null.v3"
	"testing"
)

func TestRepositoryImpl_CreateStamp(t *testing.T) {
	t.Parallel()
	repo, _, require, user := setupWithUser(t, common)

	fid, err := GenerateIconFile(repo, "stamp")
	require.NoError(err)

	t.Run("nil file id", func(t *testing.T) {
		t.Parallel()

		_, err := repo.CreateStamp(utils.RandAlphabetAndNumberString(20), uuid.Nil, user.GetID())
		assert.Error(t, err)
	})

	t.Run("invalid name", func(t *testing.T) {
		t.Parallel()

		_, err := repo.CreateStamp("あ", fid, user.GetID())
		assert.Error(t, err)
	})

	t.Run("file not found", func(t *testing.T) {
		t.Parallel()

		_, err := repo.CreateStamp(utils.RandAlphabetAndNumberString(20), uuid.Must(uuid.NewV4()), user.GetID())
		assert.Error(t, err)
	})

	t.Run("duplicate name", func(t *testing.T) {
		t.Parallel()
		s := mustMakeStamp(t, repo, random, uuid.Nil)

		_, err := repo.CreateStamp(s.Name, fid, user.GetID())
		assert.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		assert, _ := assertAndRequire(t)

		name := utils.RandAlphabetAndNumberString(20)
		s, err := repo.CreateStamp(name, fid, user.GetID())
		if assert.NoError(err) {
			assert.NotEmpty(s.ID)
			assert.Equal(name, s.Name)
			assert.Equal(fid, s.FileID)
			assert.Equal(user.GetID(), s.CreatorID)
			assert.NotEmpty(s.CreatedAt)
			assert.NotEmpty(s.UpdatedAt)
			assert.Nil(s.DeletedAt)
		}
	})
}

func TestRepositoryImpl_UpdateStamp(t *testing.T) {
	t.Parallel()
	repo, _, _ := setup(t, common)

	s := mustMakeStamp(t, repo, random, uuid.Nil)

	t.Run("nil id", func(t *testing.T) {
		t.Parallel()

		assert.EqualError(t, repo.UpdateStamp(uuid.Nil, UpdateStampArgs{}), ErrNilID.Error())
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		assert.EqualError(t, repo.UpdateStamp(uuid.Must(uuid.NewV4()), UpdateStampArgs{}), ErrNotFound.Error())
	})

	t.Run("no change", func(t *testing.T) {
		t.Parallel()

		assert.NoError(t, repo.UpdateStamp(s.ID, UpdateStampArgs{}))
	})

	t.Run("invalid name", func(t *testing.T) {
		t.Parallel()

		assert.Error(t, repo.UpdateStamp(s.ID, UpdateStampArgs{Name: null.StringFrom("あ")}))
	})

	t.Run("duplicate name", func(t *testing.T) {
		t.Parallel()

		assert.Error(t, repo.UpdateStamp(s.ID, UpdateStampArgs{Name: null.StringFrom(s.Name)}))
	})

	t.Run("nil file id", func(t *testing.T) {
		t.Parallel()

		assert.Error(t, repo.UpdateStamp(s.ID, UpdateStampArgs{FileID: uuid.NullUUID{Valid: true}}))
	})

	t.Run("file not found", func(t *testing.T) {
		t.Parallel()

		assert.Error(t, repo.UpdateStamp(s.ID, UpdateStampArgs{FileID: uuid.NullUUID{Valid: true, UUID: uuid.Must(uuid.NewV4())}}))
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		assert, require := assertAndRequire(t)

		s := mustMakeStamp(t, repo, random, uuid.Nil)
		newFile, err := GenerateIconFile(repo, "stamp")
		require.NoError(err)
		newName := utils.RandAlphabetAndNumberString(20)

		if assert.NoError(repo.UpdateStamp(s.ID, UpdateStampArgs{
			Name:      null.StringFrom(newName),
			FileID:    uuid.NullUUID{Valid: true, UUID: newFile},
			CreatorID: uuid.NullUUID{Valid: true},
		})) {
			a, err := repo.GetStamp(s.ID)
			require.NoError(err)
			assert.Equal(newFile, a.FileID)
			assert.Equal(newName, a.Name)
		}
	})
}

func TestRepositoryImpl_GetStamp(t *testing.T) {
	t.Parallel()
	repo, _, _ := setup(t, common)

	t.Run("nil id", func(t *testing.T) {
		t.Parallel()

		_, err := repo.GetStamp(uuid.Nil)
		assert.EqualError(t, err, ErrNotFound.Error())
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		_, err := repo.GetStamp(uuid.Must(uuid.NewV4()))
		assert.EqualError(t, err, ErrNotFound.Error())
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		assert, _ := assertAndRequire(t)
		a := mustMakeStamp(t, repo, random, uuid.Nil)

		s, err := repo.GetStamp(a.ID)
		if assert.NoError(err) {
			assert.Equal(a.ID, s.ID)
			assert.Equal(a.Name, s.Name)
			assert.Equal(a.FileID, s.FileID)
			assert.Equal(a.CreatorID, s.CreatorID)
		}
	})
}

func TestRepositoryImpl_DeleteStamp(t *testing.T) {
	t.Parallel()
	repo, _, _ := setup(t, common)

	t.Run("nil id", func(t *testing.T) {
		t.Parallel()

		assert.EqualError(t, repo.DeleteStamp(uuid.Nil), ErrNilID.Error())
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		assert.EqualError(t, repo.DeleteStamp(uuid.Must(uuid.NewV4())), ErrNotFound.Error())
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		assert, _ := assertAndRequire(t)

		s := mustMakeStamp(t, repo, random, uuid.Nil)
		if assert.NoError(repo.DeleteStamp(s.ID)) {
			_, err := repo.GetStamp(s.ID)
			assert.EqualError(err, ErrNotFound.Error())
		}
	})
}

func TestRepositoryImpl_GetAllStamps(t *testing.T) {
	t.Parallel()
	repo, assert, _ := setup(t, ex1)

	n := 10
	for i := 0; i < 10; i++ {
		mustMakeStamp(t, repo, random, uuid.Nil)
	}

	arr, err := repo.GetAllStamps(false)
	if assert.NoError(err) {
		assert.Len(arr, n)
	}
}

func TestRepositoryImpl_StampExists(t *testing.T) {
	t.Parallel()
	repo, _, _ := setup(t, common)

	s := mustMakeStamp(t, repo, random, uuid.Nil)

	t.Run("nil id", func(t *testing.T) {
		t.Parallel()

		ok, err := repo.StampExists(uuid.Nil)
		if assert.NoError(t, err) {
			assert.False(t, ok)
		}
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		ok, err := repo.StampExists(uuid.Must(uuid.NewV4()))
		if assert.NoError(t, err) {
			assert.False(t, ok)
		}
	})

	t.Run("found", func(t *testing.T) {
		t.Parallel()

		ok, err := repo.StampExists(s.ID)
		if assert.NoError(t, err) {
			assert.True(t, ok)
		}
	})
}

func TestRepositoryImpl_ExistStamps(t *testing.T) {
	t.Parallel()
	repo, _, _ := setup(t, common)

	stampIDs := make([]uuid.UUID, 0, 10)

	for i := 0; i < 10; i++ {
		s := mustMakeStamp(t, repo, random, uuid.Nil)
		stampIDs = append(stampIDs, s.ID)
	}

	t.Run("argument err", func(t *testing.T) {
		t.Parallel()
		assert, _ := assertAndRequire(t)

		stampIDsCopy := make([]uuid.UUID, len(stampIDs), cap(stampIDs))
		_ = copy(stampIDsCopy, stampIDs)
		if assert.True(len(stampIDsCopy) > 0) {
			stampIDsCopy[0] = uuid.Must(uuid.NewV4())
		}
		assert.Error(repo.ExistStamps(stampIDsCopy))
	})

	t.Run("sucess", func(t *testing.T) {
		t.Parallel()
		assert, _ := assertAndRequire(t)

		assert.NoError(repo.ExistStamps(stampIDs))
	})
}

func TestRepositoryImpl_StampNameExists(t *testing.T) {
	t.Parallel()
	repo, _, _ := setup(t, common)

	s := mustMakeStamp(t, repo, random, uuid.Nil)

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		ok, err := repo.StampNameExists("")
		if assert.NoError(t, err) {
			assert.False(t, ok)
		}
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		ok, err := repo.StampNameExists(utils.RandAlphabetAndNumberString(20))
		if assert.NoError(t, err) {
			assert.False(t, ok)
		}
	})

	t.Run("found", func(t *testing.T) {
		t.Parallel()

		ok, err := repo.StampNameExists(s.Name)
		if assert.NoError(t, err) {
			assert.True(t, ok)
		}
	})
}

func TestRepositoryImpl_GetUserStampHistory(t *testing.T) {
	t.Parallel()
	repo, _, _, user, channel := setupWithUserAndChannel(t, common)

	message := mustMakeMessage(t, repo, user.GetID(), channel.ID)
	stamp1 := mustMakeStamp(t, repo, random, uuid.Nil)
	stamp2 := mustMakeStamp(t, repo, random, uuid.Nil)
	stamp3 := mustMakeStamp(t, repo, random, uuid.Nil)
	mustAddMessageStamp(t, repo, message.ID, stamp1.ID, user.GetID())
	mustAddMessageStamp(t, repo, message.ID, stamp3.ID, user.GetID())
	mustAddMessageStamp(t, repo, message.ID, stamp2.ID, user.GetID())

	t.Run("Nil id", func(t *testing.T) {
		t.Parallel()
		ms, err := repo.GetUserStampHistory(uuid.Nil, 0)
		if assert.NoError(t, err) {
			assert.Empty(t, ms)
		}
	})

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		ms, err := repo.GetUserStampHistory(user.GetID(), 0)
		if assert.NoError(t, err) && assert.Len(t, ms, 3) {
			assert.Equal(t, ms[0].StampID, stamp2.ID)
			assert.Equal(t, ms[1].StampID, stamp3.ID)
			assert.Equal(t, ms[2].StampID, stamp1.ID)
		}
	})
}
