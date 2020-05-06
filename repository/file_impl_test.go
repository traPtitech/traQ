package repository

import (
	"bytes"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils/optional"
	"testing"
)

func TestGenerateIconFile(t *testing.T) {
	t.Parallel()
	repo, assert, require := setup(t, common)

	id, err := GenerateIconFile(repo, "salt")
	if assert.NoError(err) {
		meta, err := repo.GetFileMeta(id)
		require.NoError(err)
		assert.Equal(model.FileTypeIcon, meta.GetFileType())
	}

}

func TestRepositoryImpl_DeleteFile(t *testing.T) {
	t.Parallel()
	repo, _, _ := setup(t, common)

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		f := mustMakeFile(t, repo)
		if assert.NoError(t, repo.DeleteFile(f.GetID())) {
			_, err := repo.GetFileMeta(f.GetID())
			assert.EqualError(t, err, ErrNotFound.Error())
		}
	})

	t.Run("nil id", func(t *testing.T) {
		t.Parallel()

		assert.EqualError(t, repo.DeleteFile(uuid.Nil), ErrNilID.Error())
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		assert.EqualError(t, repo.DeleteFile(uuid.Must(uuid.NewV4())), ErrNotFound.Error())
	})
}

func TestRepositoryImpl_GetFileMeta(t *testing.T) {
	t.Parallel()
	repo, assert, _ := setup(t, common)

	file := mustMakeFile(t, repo)
	result, err := repo.GetFileMeta(file.GetID())
	if assert.NoError(err) {
		assert.Equal(file.GetID(), result.GetID())
	}

	_, err = repo.GetFileMeta(uuid.Nil)
	assert.Error(err)
}

func TestRepositoryImpl_SaveFile(t *testing.T) {
	t.Parallel()
	repo, assert, _ := setup(t, common)

	buf := bytes.NewBufferString("test message")
	f, err := repo.SaveFile(SaveFileArgs{
		FileName: "test.txt",
		FileSize: int64(buf.Len()),
		FileType: model.FileTypeUserFile,
		Src:      buf,
	})
	if assert.NoError(err) {
		assert.Equal("text/plain; charset=utf-8", f.GetMIMEType())
	}
}

func TestRepositoryImpl_IsFileAccessible(t *testing.T) {
	t.Parallel()
	repo, _, _, user := setupWithUser(t, common)

	t.Run("invalid args", func(t *testing.T) {
		t.Parallel()

		_, err := repo.IsFileAccessible(uuid.Nil, uuid.Nil)
		assert.EqualError(t, err, ErrNilID.Error())
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		_, err := repo.IsFileAccessible(uuid.Must(uuid.NewV4()), user.GetID())
		assert.EqualError(t, err, ErrNotFound.Error())
	})

	t.Run("Allow all", func(t *testing.T) {
		t.Parallel()
		f := mustMakeFile(t, repo)

		t.Run("any user", func(t *testing.T) {
			t.Parallel()

			ok, err := repo.IsFileAccessible(f.GetID(), uuid.Nil)
			if assert.NoError(t, err) {
				assert.True(t, ok)
			}
		})

		t.Run("user", func(t *testing.T) {
			t.Parallel()

			ok, err := repo.IsFileAccessible(f.GetID(), user.GetID())
			if assert.NoError(t, err) {
				assert.True(t, ok)
			}
		})
	})

	t.Run("Allow one", func(t *testing.T) {
		t.Parallel()

		buf := bytes.NewBufferString("test message")
		args := SaveFileArgs{
			FileName:  "test.txt",
			FileSize:  int64(buf.Len()),
			FileType:  model.FileTypeUserFile,
			CreatorID: optional.UUIDFrom(user.GetID()),
			Src:       buf,
			ACL:       ACL{},
		}
		f, err := repo.SaveFile(args)
		require.NoError(t, err)

		t.Run("any user", func(t *testing.T) {
			t.Parallel()

			ok, err := repo.IsFileAccessible(f.GetID(), uuid.Nil)
			if assert.NoError(t, err) {
				assert.False(t, ok)
			}
		})

		t.Run("allowed user", func(t *testing.T) {
			t.Parallel()

			ok, err := repo.IsFileAccessible(f.GetID(), user.GetID())
			if assert.NoError(t, err) {
				assert.True(t, ok)
			}
		})

		t.Run("denied user", func(t *testing.T) {
			t.Parallel()

			user := mustMakeUser(t, repo, random)
			ok, err := repo.IsFileAccessible(f.GetID(), user.GetID())
			if assert.NoError(t, err) {
				assert.False(t, ok)
			}
		})
	})

	t.Run("Allow two", func(t *testing.T) {
		t.Parallel()

		user2 := mustMakeUser(t, repo, random)
		buf := bytes.NewBufferString("test message")
		args := SaveFileArgs{
			FileName:  "test.txt",
			FileSize:  int64(buf.Len()),
			FileType:  model.FileTypeUserFile,
			CreatorID: optional.UUIDFrom(user.GetID()),
			Src:       buf,
			ACL:       ACL{user2.GetID(): true},
		}
		f, err := repo.SaveFile(args)
		require.NoError(t, err)

		t.Run("any user", func(t *testing.T) {
			t.Parallel()

			ok, err := repo.IsFileAccessible(f.GetID(), uuid.Nil)
			if assert.NoError(t, err) {
				assert.False(t, ok)
			}
		})

		t.Run("allowed user", func(t *testing.T) {
			t.Parallel()

			ok, err := repo.IsFileAccessible(f.GetID(), user.GetID())
			if assert.NoError(t, err) {
				assert.True(t, ok)
			}
		})

		t.Run("allowed user2", func(t *testing.T) {
			t.Parallel()

			ok, err := repo.IsFileAccessible(f.GetID(), user2.GetID())
			if assert.NoError(t, err) {
				assert.True(t, ok)
			}
		})

		t.Run("denied user", func(t *testing.T) {
			t.Parallel()

			user := mustMakeUser(t, repo, random)
			ok, err := repo.IsFileAccessible(f.GetID(), user.GetID())
			if assert.NoError(t, err) {
				assert.False(t, ok)
			}
		})
	})

	t.Run("Deny rule", func(t *testing.T) {
		t.Parallel()

		deniedUser := mustMakeUser(t, repo, random)
		buf := bytes.NewBufferString("test message")
		args := SaveFileArgs{
			FileName: "test.txt",
			FileSize: int64(buf.Len()),
			FileType: model.FileTypeUserFile,
			Src:      buf,
			ACL: ACL{
				uuid.Nil:           true,
				deniedUser.GetID(): false,
			},
		}
		f, err := repo.SaveFile(args)
		require.NoError(t, err)

		t.Run("any user", func(t *testing.T) {
			t.Parallel()

			ok, err := repo.IsFileAccessible(f.GetID(), uuid.Nil)
			if assert.NoError(t, err) {
				assert.True(t, ok)
			}
		})

		t.Run("allowed user", func(t *testing.T) {
			t.Parallel()

			ok, err := repo.IsFileAccessible(f.GetID(), user.GetID())
			if assert.NoError(t, err) {
				assert.True(t, ok)
			}
		})

		t.Run("denied user", func(t *testing.T) {
			t.Parallel()

			ok, err := repo.IsFileAccessible(f.GetID(), deniedUser.GetID())
			if assert.NoError(t, err) {
				assert.False(t, ok)
			}
		})
	})
}
