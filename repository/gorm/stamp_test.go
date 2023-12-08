package gorm

import (
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"

	"github.com/traPtitech/traQ/utils/optional"
	random2 "github.com/traPtitech/traQ/utils/random"
)

func TestRepositoryImpl_CreateStamp(t *testing.T) {
	t.Parallel()
	repo, _, _, user := setupWithUser(t, common2)

	fid := mustMakeDummyFile(t, repo).ID

	t.Run("nil file id", func(t *testing.T) {
		t.Parallel()

		_, err := repo.CreateStamp(repository.CreateStampArgs{Name: random2.AlphaNumeric(20), FileID: uuid.Nil, CreatorID: user.GetID()})
		assert.Error(t, err)
	})

	t.Run("invalid name", func(t *testing.T) {
		t.Parallel()

		_, err := repo.CreateStamp(repository.CreateStampArgs{Name: "あ", FileID: fid, CreatorID: user.GetID()})
		assert.Error(t, err)
	})

	t.Run("file not found", func(t *testing.T) {
		t.Parallel()

		_, err := repo.CreateStamp(repository.CreateStampArgs{Name: random2.AlphaNumeric(20), FileID: uuid.Must(uuid.NewV4()), CreatorID: user.GetID()})
		assert.Error(t, err)
	})

	t.Run("duplicate name", func(t *testing.T) {
		t.Parallel()
		s := mustMakeStamp(t, repo, rand, uuid.Nil)

		_, err := repo.CreateStamp(repository.CreateStampArgs{Name: s.Name, FileID: fid, CreatorID: user.GetID()})
		assert.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		name := random2.AlphaNumeric(20)
		s, err := repo.CreateStamp(repository.CreateStampArgs{Name: name, FileID: fid, CreatorID: user.GetID()})
		if assert.NoError(err) {
			assert.NotEmpty(s.ID)
			assert.Equal(name, s.Name)
			assert.Equal(fid, s.FileID)
			assert.Equal(user.GetID(), s.CreatorID)
			assert.NotEmpty(s.CreatedAt)
			assert.NotEmpty(s.UpdatedAt)
			assert.False(s.DeletedAt.Valid)
		}
	})
}

func TestRepositoryImpl_UpdateStamp(t *testing.T) {
	t.Parallel()
	repo, _, _ := setup(t, common2)

	s := mustMakeStamp(t, repo, rand, uuid.Nil)

	t.Run("nil id", func(t *testing.T) {
		t.Parallel()

		assert.EqualError(t, repo.UpdateStamp(uuid.Nil, repository.UpdateStampArgs{}), repository.ErrNilID.Error())
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		assert.EqualError(t, repo.UpdateStamp(uuid.Must(uuid.NewV4()), repository.UpdateStampArgs{}), repository.ErrNotFound.Error())
	})

	t.Run("no change", func(t *testing.T) {
		t.Parallel()

		assert.NoError(t, repo.UpdateStamp(s.ID, repository.UpdateStampArgs{}))
	})

	t.Run("invalid name", func(t *testing.T) {
		t.Parallel()

		assert.Error(t, repo.UpdateStamp(s.ID, repository.UpdateStampArgs{Name: optional.From("あ")}))
	})

	t.Run("duplicate name", func(t *testing.T) {
		t.Parallel()
		s2 := mustMakeStamp(t, repo, rand, uuid.Nil)

		assert.Error(t, repo.UpdateStamp(s.ID, repository.UpdateStampArgs{Name: optional.From(s2.Name)}))
	})

	t.Run("nil file id", func(t *testing.T) {
		t.Parallel()

		assert.Error(t, repo.UpdateStamp(s.ID, repository.UpdateStampArgs{FileID: optional.From(uuid.Nil)}))
	})

	t.Run("file not found", func(t *testing.T) {
		t.Parallel()

		assert.Error(t, repo.UpdateStamp(s.ID, repository.UpdateStampArgs{FileID: optional.From(uuid.Must(uuid.NewV4()))}))
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		assert, require := assertAndRequire(t)

		s := mustMakeStamp(t, repo, rand, uuid.Nil)
		newFile := mustMakeDummyFile(t, repo).ID
		newName := random2.AlphaNumeric(20)

		if assert.NoError(repo.UpdateStamp(s.ID, repository.UpdateStampArgs{
			Name:      optional.From(newName),
			FileID:    optional.From(newFile),
			CreatorID: optional.From(uuid.Nil),
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
	repo, _, _ := setup(t, common2)

	t.Run("nil id", func(t *testing.T) {
		t.Parallel()

		_, err := repo.GetStamp(uuid.Nil)
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		_, err := repo.GetStamp(uuid.Must(uuid.NewV4()))
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		a := mustMakeStamp(t, repo, rand, uuid.Nil)

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
	repo, _, _ := setup(t, common2)

	t.Run("nil id", func(t *testing.T) {
		t.Parallel()

		assert.EqualError(t, repo.DeleteStamp(uuid.Nil), repository.ErrNilID.Error())
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		assert.EqualError(t, repo.DeleteStamp(uuid.Must(uuid.NewV4())), repository.ErrNotFound.Error())
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		s := mustMakeStamp(t, repo, rand, uuid.Nil)
		if assert.NoError(repo.DeleteStamp(s.ID)) {
			_, err := repo.GetStamp(s.ID)
			assert.EqualError(err, repository.ErrNotFound.Error())
		}
	})
}

func TestRepositoryImpl_GetAllStampsWithThumbnail(t *testing.T) {
	t.Parallel()
	repo, assert, require := setup(t, ex1)

	n := 10

	for i := 0; i < 10; i++ {
		mustMakeStamp(t, repo, rand, uuid.Nil)
	}
	for i := 0; i < 10; i++ {
		stamp := mustMakeStamp(t, repo, rand, uuid.Nil)
		err := repo.DeleteFileMeta(stamp.FileID)
		require.NoError(err)
	}

	t.Run("without thumbnail", func(t *testing.T) {
		t.Parallel()
		arr, err := repo.GetAllStampsWithThumbnail(repository.StampTypeAll)
		if !assert.NoError(err) {
			t.FailNow()
		}
		assert.Len(arr, n*2)
		cnt := 0
		for _, s := range arr {
			if !s.HasThumbnail {
				cnt++
			}
		}
		assert.Equal(n, cnt)
	})
	t.Run("with thumbnail", func(t *testing.T) {
		t.Parallel()
		arr, err := repo.GetAllStampsWithThumbnail(repository.StampTypeAll)
		if !assert.NoError(err) {
			t.FailNow()
		}
		assert.Len(arr, n*2)
		cnt := 0
		for _, s := range arr {
			if s.HasThumbnail {
				cnt++
			}
		}
		assert.Equal(n, cnt)
	})
}

func TestRepositoryImpl_StampExists(t *testing.T) {
	t.Parallel()
	repo, _, _ := setup(t, common2)

	s := mustMakeStamp(t, repo, rand, uuid.Nil)

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
	repo, _, _ := setup(t, common2)

	stampIDs := make([]uuid.UUID, 0, 10)

	for i := 0; i < 10; i++ {
		s := mustMakeStamp(t, repo, rand, uuid.Nil)
		stampIDs = append(stampIDs, s.ID)
	}

	t.Run("argument err", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		stampIDsCopy := make([]uuid.UUID, len(stampIDs), cap(stampIDs))
		_ = copy(stampIDsCopy, stampIDs)
		if assert.True(len(stampIDsCopy) > 0) {
			stampIDsCopy[0] = uuid.Must(uuid.NewV4())
		}
		assert.Error(repo.ExistStamps(stampIDsCopy))
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		assert.NoError(repo.ExistStamps(stampIDs))
	})
}

func TestRepositoryImpl_GetUserStampHistory(t *testing.T) {
	t.Parallel()
	repo, _, _, user, channel := setupWithUserAndChannel(t, common2)
	user1 := mustMakeUser(t, repo, rand)

	message := mustMakeMessage(t, repo, user.GetID(), channel.ID)
	stamp1 := mustMakeStamp(t, repo, rand, uuid.Nil)
	stamp2 := mustMakeStamp(t, repo, rand, uuid.Nil)
	stamp3 := mustMakeStamp(t, repo, rand, uuid.Nil)
	mustAddMessageStamp(t, repo, message.ID, stamp1.ID, user.GetID())
	mustAddMessageStamp(t, repo, message.ID, stamp3.ID, user.GetID())
	mustAddMessageStamp(t, repo, message.ID, stamp2.ID, user.GetID())
	mustAddMessageStamp(t, repo, message.ID, stamp2.ID, user1.GetID())
	mustAddMessageStamp(t, repo, message.ID, stamp3.ID, user1.GetID())

	t.Run("Nil id", func(t *testing.T) {
		t.Parallel()
		ms, err := repo.GetUserStampHistory(uuid.Nil, 0)
		if assert.NoError(t, err) {
			assert.Empty(t, ms)
		}
	})

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		ms, err := repo.GetUserStampHistory(user.GetID(), -1)
		if assert.NoError(t, err) && assert.Len(t, ms, 3) {
			assert.Equal(t, ms[0].StampID, stamp2.ID)
			assert.Equal(t, ms[1].StampID, stamp3.ID)
			assert.Equal(t, ms[2].StampID, stamp1.ID)
		}
	})

	t.Run("Success (Limit 1)", func(t *testing.T) {
		t.Parallel()
		ms, err := repo.GetUserStampHistory(user.GetID(), 1)
		if assert.NoError(t, err) && assert.Len(t, ms, 1) {
			assert.Equal(t, ms[0].StampID, stamp2.ID)
		}
	})
}

func TestGormRepository_GetStampStats(t *testing.T) {
	t.Parallel()
	repo, _, _ := setup(t, common)

	t.Run("nil id", func(t *testing.T) {
		t.Parallel()

		_, err := repo.GetStampStats(uuid.Nil)
		assert.Error(t, err)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		_, err := repo.GetStampStats(uuid.Must(uuid.NewV4()))
		assert.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		channel := mustMakeChannel(t, repo, rand)
		user := mustMakeUser(t, repo, rand)
		stamp := mustMakeStamp(t, repo, rand, user.GetID())

		messages := make([]*model.Message, 15)

		for i := 0; i < 15; i++ {
			messages[i] = mustMakeMessage(t, repo, user.GetID(), channel.ID)
		}

		for i := 0; i < 15; i++ {
			for j := 0; j < 3; j++ {
				mustAddMessageStamp(t, repo, messages[i].ID, stamp.ID, user.GetID())
			}
		}

		stats, err := repo.GetStampStats(stamp.ID)
		if assert.NoError(t, err) {
			assert.EqualValues(t, 15, stats.Count)
			assert.EqualValues(t, 45, stats.TotalCount)
		}
	})

}
