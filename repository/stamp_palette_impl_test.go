package repository

import (
	"github.com/gofrs/uuid"
	"testing"
)

func TestRepositoryImpl_ExistStamps(t *testing.T) {
	t.Parallel()
	repo, assert, _ := setup(t, common)

	stampIDs := make([]uuid.UUID, 10)

	for i := 0; i < 10; i++ {
		s := mustMakeStamp(t, repo, random, uuid.Nil)
		stampIDs = append(stampIDs, s.ID)
	}

	t.Run("argument err", func(t *testing.T) {
		t.Parallel()

		stampIDsCopy := make([]uuid.UUID, 10)
		copy(stampIDsCopy, stampIDs)
		stampIDsCopy[0] = uuid.Must(uuid.NewV4())
		assert.Error(repo.ExistStamps(stampIDsCopy))
	})

	t.Run("sucess", func(t *testing.T) {
		t.Parallel()

		assert.NoError(repo.ExistStamps(stampIDs))
	})
}