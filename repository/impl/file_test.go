package impl

import (
	"bytes"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
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
		assert.Equal("text/plain; charset=utf-8", f.Mime)
	}
}

func TestRepositoryImpl_IsFileAccessible(t *testing.T) {
	t.Parallel()
	repo, _, _, user := setupWithUser(t, common)

	t.Run("invalid args", func(t *testing.T) {
		t.Parallel()

		_, err := repo.IsFileAccessible(uuid.Nil, uuid.Nil)
		assert.Error(t, err)
	})

	t.Run("Allow all", func(t *testing.T) {
		t.Parallel()
		f := mustMakeFile(t, repo, uuid.Nil)

		t.Run("any user", func(t *testing.T) {
			t.Parallel()

			ok, err := repo.IsFileAccessible(f.ID, uuid.Nil)
			if assert.NoError(t, err) {
				assert.True(t, ok)
			}
		})

		t.Run("user", func(t *testing.T) {
			t.Parallel()

			ok, err := repo.IsFileAccessible(f.ID, user.ID)
			if assert.NoError(t, err) {
				assert.True(t, ok)
			}
		})
	})

	t.Run("Allow one", func(t *testing.T) {
		t.Parallel()

		buf := bytes.NewBufferString("test message")
		f, err := repo.SaveFileWithACL("test.txt", buf, int64(buf.Len()), "", model.FileTypeUserFile, user.ID, repository.ACL{})
		require.NoError(t, err)

		t.Run("any user", func(t *testing.T) {
			t.Parallel()

			ok, err := repo.IsFileAccessible(f.ID, uuid.Nil)
			if assert.NoError(t, err) {
				assert.False(t, ok)
			}
		})

		t.Run("allowed user", func(t *testing.T) {
			t.Parallel()

			ok, err := repo.IsFileAccessible(f.ID, user.ID)
			if assert.NoError(t, err) {
				assert.True(t, ok)
			}
		})

		t.Run("denied user", func(t *testing.T) {
			t.Parallel()

			user := mustMakeUser(t, repo, random)
			ok, err := repo.IsFileAccessible(f.ID, user.ID)
			if assert.NoError(t, err) {
				assert.False(t, ok)
			}
		})
	})

	t.Run("Allow two", func(t *testing.T) {
		t.Parallel()

		user2 := mustMakeUser(t, repo, random)
		buf := bytes.NewBufferString("test message")
		f, err := repo.SaveFileWithACL("test.txt", buf, int64(buf.Len()), "", model.FileTypeUserFile, user.ID, repository.ACL{user2.ID: true})
		require.NoError(t, err)

		t.Run("any user", func(t *testing.T) {
			t.Parallel()

			ok, err := repo.IsFileAccessible(f.ID, uuid.Nil)
			if assert.NoError(t, err) {
				assert.False(t, ok)
			}
		})

		t.Run("allowed user", func(t *testing.T) {
			t.Parallel()

			ok, err := repo.IsFileAccessible(f.ID, user.ID)
			if assert.NoError(t, err) {
				assert.True(t, ok)
			}
		})

		t.Run("allowed user2", func(t *testing.T) {
			t.Parallel()

			ok, err := repo.IsFileAccessible(f.ID, user2.ID)
			if assert.NoError(t, err) {
				assert.True(t, ok)
			}
		})

		t.Run("denied user", func(t *testing.T) {
			t.Parallel()

			user := mustMakeUser(t, repo, random)
			ok, err := repo.IsFileAccessible(f.ID, user.ID)
			if assert.NoError(t, err) {
				assert.False(t, ok)
			}
		})
	})

	t.Run("Deny rule", func(t *testing.T) {
		t.Parallel()

		deninedUser := mustMakeUser(t, repo, random)
		buf := bytes.NewBufferString("test message")
		f, err := repo.SaveFileWithACL("test.txt", buf, int64(buf.Len()), "", model.FileTypeUserFile, uuid.Nil, repository.ACL{
			uuid.Nil:       true,
			deninedUser.ID: false,
		})
		require.NoError(t, err)

		t.Run("any user", func(t *testing.T) {
			t.Parallel()

			ok, err := repo.IsFileAccessible(f.ID, uuid.Nil)
			if assert.NoError(t, err) {
				assert.True(t, ok)
			}
		})

		t.Run("allowed user", func(t *testing.T) {
			t.Parallel()

			ok, err := repo.IsFileAccessible(f.ID, user.ID)
			if assert.NoError(t, err) {
				assert.True(t, ok)
			}
		})

		t.Run("denied user", func(t *testing.T) {
			t.Parallel()

			ok, err := repo.IsFileAccessible(f.ID, deninedUser.ID)
			if assert.NoError(t, err) {
				assert.False(t, ok)
			}
		})
	})
}
