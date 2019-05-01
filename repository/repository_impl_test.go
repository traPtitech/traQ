package repository

import (
	"github.com/stretchr/testify/assert"
	"github.com/traPtitech/traQ/utils/storage"
	"testing"
)

func TestRepositoryImpl_GetFS(t *testing.T) {
	t.Parallel()
	fs := storage.NewInMemoryFileStorage()
	repo := &RepositoryImpl{fileImpl: fileImpl{FS: fs}}
	assert.Equal(t, fs, repo.GetFS())
}
