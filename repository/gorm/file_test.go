package gorm

import (
	"slices"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
)

func TestGormRepository_SaveFileMeta(t *testing.T) {
	t.Parallel()
	repo, _, _ := setup(t, common)

	t.Run("nil file", func(t *testing.T) {
		t.Parallel()

		assert.Error(t, repo.SaveFileMeta(nil, nil))
	})

	t.Run("success with UUIDv4", func(t *testing.T) {
		t.Parallel()
		meta := &model.FileMeta{
			ID:   uuid.Must(uuid.NewV4()),
			Name: "dummy",
			Mime: "application/octet-stream",
			Size: 10,
			Hash: "d41d8cd98f00b204e9800998ecf8427e",
			Type: model.FileTypeUserFile,
		}
		acl := []*model.FileACLEntry{
			{UserID: uuid.Nil, Allow: true},
		}

		err := repo.SaveFileMeta(meta, acl)
		if assert.NoError(t, err) {
			assert.NotEmpty(t, meta.CreatedAt)
			assert.False(t, meta.DeletedAt.Valid)
		}
	})
	t.Run("success with UUIDv7", func(t *testing.T) {
		t.Parallel()
		meta := &model.FileMeta{
			ID:   uuid.Must(uuid.NewV7()),
			Name: "dummy",
			Mime: "application/octet-stream",
			Size: 10,
			Hash: "d41d8cd98f00b204e9800998ecf8427e",
			Type: model.FileTypeUserFile,
		}
		acl := []*model.FileACLEntry{
			{UserID: uuid.Nil, Allow: true},
		}

		err := repo.SaveFileMeta(meta, acl)
		if assert.NoError(t, err) {
			assert.NotEmpty(t, meta.CreatedAt)
			assert.False(t, meta.DeletedAt.Valid)
		}
	})
}

func TestGormRepository_GetFileMeta(t *testing.T) {
	t.Parallel()
	repo, _, _ := setup(t, common)

	f := mustMakeDummyFile(t, repo)

	t.Run("nil id", func(t *testing.T) {
		t.Parallel()

		_, err := repo.GetFileMeta(uuid.Nil)
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		_, err := repo.GetFileMeta(uuid.NewV3(uuid.Nil, "not found"))
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})

	t.Run("found", func(t *testing.T) {
		t.Parallel()

		meta, err := repo.GetFileMeta(f.ID)
		if assert.NoError(t, err) {
			assert.EqualValues(t, f.ID, meta.ID)
			assert.EqualValues(t, 1, len(meta.Thumbnails))
			assert.EqualValues(t, f.ID, meta.Thumbnails[0].FileID)
		}
	})
}

func TestGormRepository_DeleteFileMeta(t *testing.T) {
	t.Parallel()
	repo, _, _ := setup(t, common)

	t.Run("nil id", func(t *testing.T) {
		t.Parallel()

		err := repo.DeleteFileMeta(uuid.Nil)
		assert.EqualError(t, err, repository.ErrNilID.Error())
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		f := mustMakeDummyFile(t, repo)

		err := repo.DeleteFileMeta(f.ID)
		if assert.NoError(t, err) {
			assert.Equal(t, 0, count(t, getDB(repo).Model(&model.FileMeta{}).Where(&model.FileMeta{ID: f.ID})))
		}
	})

	t.Run("success (noop)", func(t *testing.T) {
		t.Parallel()

		err := repo.DeleteFileMeta(uuid.NewV3(uuid.Nil, "not exists"))
		assert.NoError(t, err)
	})
}

func TestGormRepository_IsFileAccessible(t *testing.T) {
	t.Parallel()
	repo, _, _, user := setupWithUser(t, common)

	t.Run("file which doesn't exist", func(t *testing.T) {
		t.Parallel()

		ok, err := repo.IsFileAccessible(uuid.NewV3(uuid.Nil, "not exists"), uuid.Nil)
		if assert.NoError(t, err) {
			assert.False(t, ok)
		}
	})

	t.Run("nil id", func(t *testing.T) {
		t.Parallel()

		ok, err := repo.IsFileAccessible(uuid.Nil, uuid.Nil)
		if assert.NoError(t, err) {
			assert.False(t, ok)
		}
	})

	t.Run("allow everyone", func(t *testing.T) {
		t.Parallel()
		f := mustMakeDummyFile(t, repo)

		t.Run("any users", func(t *testing.T) {
			t.Parallel()

			ok, err := repo.IsFileAccessible(f.ID, uuid.Nil)
			if assert.NoError(t, err) {
				assert.True(t, ok)
			}
		})

		t.Run("a certain user", func(t *testing.T) {
			t.Parallel()

			ok, err := repo.IsFileAccessible(f.ID, uuid.NewV3(uuid.Nil, "u1"))
			if assert.NoError(t, err) {
				assert.True(t, ok)
			}
		})
	})

	t.Run("allow one(UUIDv4)", func(t *testing.T) {
		t.Parallel()

		meta := &model.FileMeta{
			ID:   uuid.Must(uuid.NewV4()),
			Name: "dummy",
			Mime: "application/octet-stream",
			Size: 10,
			Hash: "d41d8cd98f00b204e9800998ecf8427e",
			Type: model.FileTypeUserFile,
		}
		err := repo.SaveFileMeta(meta, []*model.FileACLEntry{
			{UserID: user.GetID(), Allow: true},
		})
		require.NoError(t, err)

		t.Run("any users", func(t *testing.T) {
			t.Parallel()

			ok, err := repo.IsFileAccessible(meta.ID, uuid.Nil)
			if assert.NoError(t, err) {
				assert.False(t, ok)
			}
		})

		t.Run("allowed user", func(t *testing.T) {
			t.Parallel()

			ok, err := repo.IsFileAccessible(meta.ID, user.GetID())
			if assert.NoError(t, err) {
				assert.True(t, ok)
			}
		})

		t.Run("denied user", func(t *testing.T) {
			t.Parallel()

			user := mustMakeUser(t, repo, rand)
			ok, err := repo.IsFileAccessible(meta.ID, user.GetID())
			if assert.NoError(t, err) {
				assert.False(t, ok)
			}
		})
	})

	t.Run("allow one(UUIDv7)", func(t *testing.T) {
		t.Parallel()

		meta := &model.FileMeta{
			ID:   uuid.Must(uuid.NewV7()),
			Name: "dummy",
			Mime: "application/octet-stream",
			Size: 10,
			Hash: "d41d8cd98f00b204e9800998ecf8427e",
			Type: model.FileTypeUserFile,
		}
		err := repo.SaveFileMeta(meta, []*model.FileACLEntry{
			{UserID: user.GetID(), Allow: true},
		})
		require.NoError(t, err)

		t.Run("any users", func(t *testing.T) {
			t.Parallel()

			ok, err := repo.IsFileAccessible(meta.ID, uuid.Nil)
			if assert.NoError(t, err) {
				assert.False(t, ok)
			}
		})

		t.Run("allowed user", func(t *testing.T) {
			t.Parallel()

			ok, err := repo.IsFileAccessible(meta.ID, user.GetID())
			if assert.NoError(t, err) {
				assert.True(t, ok)
			}
		})

		t.Run("denied user", func(t *testing.T) {
			t.Parallel()

			user := mustMakeUser(t, repo, rand)
			ok, err := repo.IsFileAccessible(meta.ID, user.GetID())
			if assert.NoError(t, err) {
				assert.False(t, ok)
			}
		})
	})

	t.Run("allow two(UUIDv4)", func(t *testing.T) {
		t.Parallel()

		user2 := mustMakeUser(t, repo, rand)
		meta := &model.FileMeta{
			ID:   uuid.Must(uuid.NewV4()),
			Name: "dummy",
			Mime: "application/octet-stream",
			Size: 10,
			Hash: "d41d8cd98f00b204e9800998ecf8427e",
			Type: model.FileTypeUserFile,
		}
		err := repo.SaveFileMeta(meta, []*model.FileACLEntry{
			{UserID: user.GetID(), Allow: true},
			{UserID: user2.GetID(), Allow: true},
		})
		require.NoError(t, err)

		t.Run("any users", func(t *testing.T) {
			t.Parallel()

			ok, err := repo.IsFileAccessible(meta.ID, uuid.Nil)
			if assert.NoError(t, err) {
				assert.False(t, ok)
			}
		})

		t.Run("allowed user", func(t *testing.T) {
			t.Parallel()

			ok, err := repo.IsFileAccessible(meta.ID, user.GetID())
			if assert.NoError(t, err) {
				assert.True(t, ok)
			}
		})

		t.Run("allowed user2", func(t *testing.T) {
			t.Parallel()

			ok, err := repo.IsFileAccessible(meta.ID, user2.GetID())
			if assert.NoError(t, err) {
				assert.True(t, ok)
			}
		})

		t.Run("denied user", func(t *testing.T) {
			t.Parallel()

			user := mustMakeUser(t, repo, rand)
			ok, err := repo.IsFileAccessible(meta.ID, user.GetID())
			if assert.NoError(t, err) {
				assert.False(t, ok)
			}
		})
	})
	t.Run("allow two(UUIDv7)", func(t *testing.T) {
		t.Parallel()

		user2 := mustMakeUser(t, repo, rand)
		meta := &model.FileMeta{
			ID:   uuid.Must(uuid.NewV7()),
			Name: "dummy",
			Mime: "application/octet-stream",
			Size: 10,
			Hash: "d41d8cd98f00b204e9800998ecf8427e",
			Type: model.FileTypeUserFile,
		}
		err := repo.SaveFileMeta(meta, []*model.FileACLEntry{
			{UserID: user.GetID(), Allow: true},
			{UserID: user2.GetID(), Allow: true},
		})
		require.NoError(t, err)

		t.Run("any users", func(t *testing.T) {
			t.Parallel()

			ok, err := repo.IsFileAccessible(meta.ID, uuid.Nil)
			if assert.NoError(t, err) {
				assert.False(t, ok)
			}
		})

		t.Run("allowed user", func(t *testing.T) {
			t.Parallel()

			ok, err := repo.IsFileAccessible(meta.ID, user.GetID())
			if assert.NoError(t, err) {
				assert.True(t, ok)
			}
		})

		t.Run("allowed user2", func(t *testing.T) {
			t.Parallel()

			ok, err := repo.IsFileAccessible(meta.ID, user2.GetID())
			if assert.NoError(t, err) {
				assert.True(t, ok)
			}
		})

		t.Run("denied user", func(t *testing.T) {
			t.Parallel()

			user := mustMakeUser(t, repo, rand)
			ok, err := repo.IsFileAccessible(meta.ID, user.GetID())
			if assert.NoError(t, err) {
				assert.False(t, ok)
			}
		})
	})

	t.Run("deny rule(UUIDv4)", func(t *testing.T) {
		t.Parallel()

		deniedUser := mustMakeUser(t, repo, rand)
		meta := &model.FileMeta{
			ID:   uuid.Must(uuid.NewV4()),
			Name: "dummy",
			Mime: "application/octet-stream",
			Size: 10,
			Hash: "d41d8cd98f00b204e9800998ecf8427e",
			Type: model.FileTypeUserFile,
		}
		err := repo.SaveFileMeta(meta, []*model.FileACLEntry{
			{UserID: uuid.Nil, Allow: true},
			{UserID: deniedUser.GetID(), Allow: false},
		})
		require.NoError(t, err)

		t.Run("any user", func(t *testing.T) {
			t.Parallel()

			ok, err := repo.IsFileAccessible(meta.ID, uuid.Nil)
			if assert.NoError(t, err) {
				assert.True(t, ok)
			}
		})

		t.Run("allowed user", func(t *testing.T) {
			t.Parallel()

			ok, err := repo.IsFileAccessible(meta.ID, user.GetID())
			if assert.NoError(t, err) {
				assert.True(t, ok)
			}
		})

		t.Run("denied user", func(t *testing.T) {
			t.Parallel()

			ok, err := repo.IsFileAccessible(meta.ID, deniedUser.GetID())
			if assert.NoError(t, err) {
				assert.False(t, ok)
			}
		})
	})

	t.Run("deny rule(UUIDv7)", func(t *testing.T) {
		t.Parallel()

		deniedUser := mustMakeUser(t, repo, rand)
		meta := &model.FileMeta{
			ID:   uuid.Must(uuid.NewV7()),
			Name: "dummy",
			Mime: "application/octet-stream",
			Size: 10,
			Hash: "d41d8cd98f00b204e9800998ecf8427e",
			Type: model.FileTypeUserFile,
		}
		err := repo.SaveFileMeta(meta, []*model.FileACLEntry{
			{UserID: uuid.Nil, Allow: true},
			{UserID: deniedUser.GetID(), Allow: false},
		})
		require.NoError(t, err)

		t.Run("any user", func(t *testing.T) {
			t.Parallel()

			ok, err := repo.IsFileAccessible(meta.ID, uuid.Nil)
			if assert.NoError(t, err) {
				assert.True(t, ok)
			}
		})

		t.Run("allowed user", func(t *testing.T) {
			t.Parallel()

			ok, err := repo.IsFileAccessible(meta.ID, user.GetID())
			if assert.NoError(t, err) {
				assert.True(t, ok)
			}
		})

		t.Run("denied user", func(t *testing.T) {
			t.Parallel()

			ok, err := repo.IsFileAccessible(meta.ID, deniedUser.GetID())
			if assert.NoError(t, err) {
				assert.False(t, ok)
			}
		})
	})
}

// model.FileThumbnail の Type が指定されたものと等価であるか比較する関数を取得する.
func getThumbnailEqualityComparerByType(tt model.ThumbnailType) func(model.FileThumbnail) bool {
	return func(t model.FileThumbnail) bool {
		return t.Type == tt
	}
}

func TestGormRepository_DeleteFileThumbnail(t *testing.T) {
	t.Parallel()
	repo, _, _ := setup(t, common)

	tests := map[string]struct {
		createsFile           bool
		deletesExistingFile   bool
		thumbnailTypeToDelete model.ThumbnailType
		uuidVersion           string // "v4" or "v7"
	}{
		"nil id": {
			createsFile:           false,
			deletesExistingFile:   false,
			thumbnailTypeToDelete: model.ThumbnailTypeImage,
			uuidVersion:           "v7",
		},
		"file not found (UUIDv4)": {
			createsFile:           true,
			deletesExistingFile:   false,
			thumbnailTypeToDelete: model.ThumbnailTypeImage,
			uuidVersion:           "v4",
		},
		"file not found (UUIDv7)": {
			createsFile:           true,
			deletesExistingFile:   false,
			thumbnailTypeToDelete: model.ThumbnailTypeImage,
			uuidVersion:           "v7",
		},
		"thumbnail type not found (UUIDv4)": {
			createsFile:           true,
			deletesExistingFile:   true,
			thumbnailTypeToDelete: model.ThumbnailTypeWaveform,
			uuidVersion:           "v4",
		},
		"thumbnail type not found (UUIDv7)": {
			createsFile:           true,
			deletesExistingFile:   true,
			thumbnailTypeToDelete: model.ThumbnailTypeWaveform,
			uuidVersion:           "v7",
		},
		"success (UUIDv4)": {
			createsFile:           true,
			deletesExistingFile:   true,
			thumbnailTypeToDelete: model.ThumbnailTypeImage,
			uuidVersion:           "v4",
		},
		"success (UUIDv7)": {
			createsFile:           true,
			deletesExistingFile:   true,
			thumbnailTypeToDelete: model.ThumbnailTypeImage,
			uuidVersion:           "v7",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if !tt.createsFile {
				err := repo.DeleteFileThumbnail(uuid.Nil, tt.thumbnailTypeToDelete)
				assert.EqualError(t, err, repository.ErrNilID.Error())
				return
			}

			f := mustMakeDummyFile(t, repo)

			if !tt.deletesExistingFile { // 存在しないファイルの場合, 変更なしを検証
				var nonExistentID uuid.UUID
				if tt.uuidVersion == "v4" {
					nonExistentID = uuid.Must(uuid.NewV4())
				} else {
					nonExistentID = uuid.Must(uuid.NewV7())
				}
				err := repo.DeleteFileThumbnail(nonExistentID, tt.thumbnailTypeToDelete)
				assert.NoError(t, err)
				ff, err := repo.GetFileMeta(f.ID)
				assert.NoError(t, err)
				assert.ElementsMatch(t, f.Thumbnails, ff.Thumbnails)
				return
			}

			err := repo.DeleteFileThumbnail(f.ID, tt.thumbnailTypeToDelete)
			assert.NoError(t, err)
			if !slices.ContainsFunc(f.Thumbnails, getThumbnailEqualityComparerByType(tt.thumbnailTypeToDelete)) { // f.Thumbnails が tt.thumbnailTypeToDelete を含まない場合, 変更なしを検証
				ff, err := repo.GetFileMeta(f.ID)
				assert.NoError(t, err)
				assert.ElementsMatch(t, f.Thumbnails, ff.Thumbnails)
				return
			}
			f, err = repo.GetFileMeta(f.ID)
			assert.NoError(t, err)
			assert.False(t, slices.ContainsFunc(f.Thumbnails, getThumbnailEqualityComparerByType(tt.thumbnailTypeToDelete))) // f.Thumbnails が tt.thumbnailTypeToDelete を含まない
		})
	}
}
