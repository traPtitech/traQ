package impl

import (
	"bytes"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/model"
	"testing"
)

func TestRepositoryImpl_DeleteFile(t *testing.T) {
	t.Parallel()
	repo, assert, _ := setup(t, common)

	file := mustMakeFile(t, repo, uuid.Nil)
	if assert.NoError(repo.DeleteFile(file.ID)) {
		_, err := repo.GetFileMeta(file.ID)
		assert.Error(err)
	}
}

func TestRepositoryImpl_OpenFile(t *testing.T) {
	t.Parallel()
	repo, assert, _ := setup(t, common)

	f := mustMakeFile(t, repo, uuid.Nil)
	_, file, err := repo.OpenFile(f.ID)
	if assert.NoError(err) {
		buf := make([]byte, 512)
		n, err := file.Read(buf)
		_ = file.Close()
		if assert.NoError(err) {
			assert.Equal("test message", string(buf[:n]))
		}
	}
}

func TestRepositoryImpl_GetFileMeta(t *testing.T) {
	t.Parallel()
	repo, assert, _ := setup(t, common)

	file := mustMakeFile(t, repo, uuid.Nil)
	result, err := repo.GetFileMeta(file.ID)
	if assert.NoError(err) {
		assert.Equal(file.ID, result.ID)
	}

	_, err = repo.GetFileMeta(uuid.Nil)
	assert.Error(err)
}

func TestRepositoryImpl_SaveFile(t *testing.T) {
	t.Parallel()
	repo, assert, _ := setup(t, common)

	buf := bytes.NewBufferString("test message")
	f, err := repo.SaveFile("test.txt", buf, int64(buf.Len()), "", model.FileTypeUserFile, uuid.Nil)
	if assert.NoError(err) {
		assert.Equal("text/plain", f.Mime)
	}
}
